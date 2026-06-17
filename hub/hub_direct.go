package hub

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/turbot/steampipe-plugin-sdk/v6/grpc/proto"
	"github.com/turbot/steampipe-postgres-fdw/v2/settings"
	"github.com/turbot/steampipe-postgres-fdw/v2/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type HubDirect struct {
	hubBase
	conn    *grpc.ClientConn
	client  proto.WrapperPluginClient
	addr    string
	schemas map[string]*proto.Schema // cached per connection
}

func newDirectHub(addr string) (*HubDirect, error) {
	log.Printf("[INFO] newDirectHub connecting to %s", addr)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server at %s: %w", addr, err)
	}

	hub := &HubDirect{
		hubBase: newHubBase(true),
		conn:    conn,
		client:  proto.NewWrapperPluginClient(conn),
		addr:    addr,
		schemas: make(map[string]*proto.Schema),
	}
	hub.cacheSettings = settings.NewCacheSettings(nil, false)

	if err := hub.initialiseTelemetry(); err != nil {
		return nil, err
	}

	return hub, nil
}

func (h *HubDirect) GetConnectionConfigByName(name string) *proto.ConnectionConfig {
	return nil
}

func (h *HubDirect) LoadConnectionConfig() (bool, error) {
	return false, nil
}

func (h *HubDirect) GetSchema(remoteSchema string, connectionName string) (*proto.Schema, error) {
	if cached, ok := h.schemas[connectionName]; ok {
		return cached, nil
	}

	resp, err := h.client.GetSchema(context.Background(), &proto.GetSchemaRequest{
		Connection: connectionName,
	})
	if err != nil {
		return nil, fmt.Errorf("gRPC GetSchema for connection %s: %w", connectionName, err)
	}

	h.schemas[connectionName] = resp.Schema
	return resp.Schema, nil
}

func (h *HubDirect) GetIterator(columns []string, quals *proto.Quals, unhandledRestrictions int, limit int64, sortOrder []*proto.SortColumn, queryTimestamp int64, opts types.Options) (Iterator, error) {
	connectionName := opts["connection"]
	table := opts["table"]

	log.Printf("[INFO] HubDirect.GetIterator connection=%s table=%s", connectionName, table)

	connectionSchema, err := h.GetSchema("", connectionName)
	if err != nil {
		return nil, err
	}

	qualMap := map[string]*proto.Quals{}
	if quals != nil && quals.Quals != nil {
		qualMap, err = buildQualMap(quals)
	}

	// Determine limit pushdown
	var resolvedLimit int64 = -1
	if limit > 0 && h.shouldPushdownLimit(table, qualMap, unhandledRestrictions, connectionSchema) {
		resolvedLimit = limit
	}

	connectionLimitMap := map[string]int64{connectionName: resolvedLimit}

	traceCtx := h.traceContextForScan(table, columns, limit, qualMap, connectionName, opts)

	iterator := newScanIteratorDirect(h, connectionName, table, connectionLimitMap, qualMap, columns, resolvedLimit, sortOrder, queryTimestamp, traceCtx)
	return iterator, nil
}

func (h *HubDirect) GetPathKeys(opts types.Options) ([]types.PathKey, error) {
	connectionName := opts["connection"]
	connectionSchema, err := h.GetSchema("", connectionName)
	if err != nil {
		return nil, err
	}
	return h.getPathKeys(connectionSchema, opts)
}

func (h *HubDirect) Close() {
	h.hubBase.Close()
	if h.conn != nil {
		h.conn.Close()
	}
}

func (h *HubDirect) cacheTTL(_ string) time.Duration {
	return 0
}

func (h *HubDirect) cacheEnabled(_ string) bool {
	return false
}
