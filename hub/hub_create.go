package hub

import (
	"os"
	"sync"

	"github.com/turbot/steampipe-plugin-sdk/v6/logging"
)

// global hub instance
var hubSingleton Hub

// mutex protecting hub creation
var hubMux sync.Mutex

// GetHub returns a hub singleton
func GetHub() Hub {
	// lock access to singleton
	hubMux.Lock()
	defer hubMux.Unlock()
	return hubSingleton
}

// CreateHub creates the hub
func CreateHub() error {
	logging.LogTime("GetHub start")

	// lock access to singleton
	hubMux.Lock()
	defer hubMux.Unlock()

	var err error
	if addr := os.Getenv("STEAMPIPE_GRPC_ADDRESS"); addr != "" {
		hubSingleton, err = newDirectHub(addr)
	} else {
		hubSingleton, err = newRemoteHub()
	}
	if err != nil {
		return err
	}
	logging.LogTime("GetHub end")
	return err
}
