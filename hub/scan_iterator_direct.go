package hub

import (
	"context"
	"log"

	"github.com/turbot/steampipe-plugin-sdk/v6/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v6/row_stream"
	"github.com/turbot/steampipe-plugin-sdk/v6/telemetry"
)

type scanIteratorDirect struct {
	scanIteratorBase
	pluginName string
}

func newScanIteratorDirect(hub *HubDirect, connectionName, table string, connectionLimitMap map[string]int64, qualMap map[string]*proto.Quals, columns []string, limit int64, sortOrder []*proto.SortColumn, queryTimestamp int64, traceCtx *telemetry.TraceCtx) *scanIteratorDirect {
	return &scanIteratorDirect{
		scanIteratorBase: newBaseScanIterator(hub, connectionName, table, connectionLimitMap, qualMap, columns, limit, sortOrder, queryTimestamp, traceCtx),
		pluginName:       "direct",
	}
}

func (i *scanIteratorDirect) GetPluginName() string {
	return i.pluginName
}

func (i *scanIteratorDirect) execute(req *proto.ExecuteRequest) (row_stream.Receiver, context.Context, context.CancelFunc, error) {
	hub := i.hub.(*HubDirect)

	ctx, cancel := context.WithCancel(context.Background())
	log.Printf("[INFO] scanIteratorDirect.execute table=%s connection=%s callId=%s", i.table, i.connectionName, i.callId)

	stream, err := hub.client.Execute(ctx, req)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}

	// WrapperPlugin_ExecuteClient implements row_stream.Receiver (Recv() (*ExecuteResponse, error))
	return stream, ctx, cancel, nil
}
