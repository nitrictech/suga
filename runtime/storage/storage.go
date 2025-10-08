package storage

import (
	"fmt"

	storagepb "github.com/nitrictech/suga/proto/storage/v2"
	"github.com/nitrictech/suga/runtime/env"
	"github.com/nitrictech/suga/runtime/plugin"
)

// Define the interface for a storage plugin here
type Storage = storagepb.StorageServer

// Available storage plugins for runtime
var storagePlugins = make(map[string]Storage)

func GetPluginByResourceName(name string) (Storage, error) {
	sugaResourceTypes := env.GetSugaResourceTypes()
	bucketTypes := sugaResourceTypes["bucket"]

	if bucketTypes == nil {
		return nil, fmt.Errorf("no bucket types found in environment")
	}

	pluginNamespace, ok := bucketTypes[name]
	if !ok {
		return nil, fmt.Errorf("no plugin found for bucket type: %s", name)
	}

	return GetPlugin(pluginNamespace), nil
}

func GetPlugin(namespace string) Storage {
	return storagePlugins[namespace]
}

// Register a new instance of a storage plugin
func Register(namespace string, constructor plugin.Constructor[Storage]) error {
	storagePlugin, err := constructor()
	if err != nil {
		return err
	}

	storagePlugins[namespace] = storagePlugin
	return nil
}
