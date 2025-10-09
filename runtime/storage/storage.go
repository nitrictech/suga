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
	sugaTypes, err := env.GetSugaResourceTypes()
	if err != nil {
		return nil, fmt.Errorf("could not resolve access plugin for resource %q: %w", name, err)
	}

	bucketTypes, ok := sugaTypes["bucket"]
	if !ok || len(bucketTypes) == 0 {
		return nil, fmt.Errorf("could not resolve access plugin for resource %q: no bucket plugin mappings found in environment", name)
	}

	pluginNamespace, ok := bucketTypes[name]
	if !ok {
		return nil, fmt.Errorf("no plugin mapping found for bucket: %s", name)
	}

	return GetPlugin(pluginNamespace)
}

func GetPlugin(namespace string) (Storage, error) {
	plugin, ok := storagePlugins[namespace]
	if !ok {
		return nil, fmt.Errorf("no storage plugin registered for namespace: %s", namespace)
	}
	return plugin, nil
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
