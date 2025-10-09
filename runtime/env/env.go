package env

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var (
	sugaResourceTypes     map[string]map[string]string
	sugaResourceTypesOnce sync.Once
	sugaResourceTypesErr  error
)

func GetSugaResourceTypes() (map[string]map[string]string, error) {
	sugaResourceTypesOnce.Do(func() {
		resourceTypes := os.Getenv("SUGA_RESOURCE_TYPES")

		if resourceTypes == "" {
			sugaResourceTypesErr = fmt.Errorf("SUGA_RESOURCE_TYPES environment variable is not set, this indicates an issue with the Suga platform used for deployment")
			return
		}

		var result map[string]map[string]string
		if err := json.Unmarshal([]byte(resourceTypes), &result); err != nil {
			sugaResourceTypesErr = fmt.Errorf("failed to unmarshal SUGA_RESOURCE_TYPES: %w, this may indicate an issue with Suga platform used for deployment", err)
			return
		}

		sugaResourceTypes = result
	})

	return sugaResourceTypes, sugaResourceTypesErr
}
