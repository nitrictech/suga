package client

import (
	"github.com/nitrictech/suga/cli/pkg/schema"
)

func SpecHasClientResources(appSpec schema.Application) bool {
	for _, intent := range appSpec.GetResourceIntents() {
		// TODO: Add other adaptable resources here.
		if intent.GetType() == "bucket" {
			return true
		}
	}

	return false
}
