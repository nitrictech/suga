package terraform

type SugaVariables struct {
	Name *string `json:"name"`
}

type SugaServiceVariables struct {
	SugaVariables `json:",inline"`
	ImageId       *string                         `json:"image_id"`
	Env           interface{}                     `json:"env"`
	Identities    *map[string]interface{}         `json:"identities"`
	Services      map[string]interface{}          `json:"services,omitempty"`
	Schedules     *map[string]SugaServiceSchedule `json:"schedules,omitempty"`
	StackId       *string                         `json:"stack_id"`
}

type SugaServiceSchedule struct {
	CronExpression *string `json:"cron_expression"`
	Path           *string `json:"path"`
}

type SugaOutputs struct {
	Id *string `json:"id"`
}

type SugaServiceOutputs struct {
	SugaOutputs  `json:",inline"`
	HttpEndpoint *string `json:"http_endpoint"`
}
