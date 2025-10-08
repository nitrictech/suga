package env

import (
	"encoding/json"
	"os"
)

var sugaResourceTypes map[string]map[string]string

func GetSugaResourceTypes() map[string]map[string]string {
	if sugaResourceTypes == nil {
		resourceTypes := os.Getenv("SUGA_RESOURCE_TYPES")

		json.Unmarshal([]byte(resourceTypes), &sugaResourceTypes)
	}

	return sugaResourceTypes
}
