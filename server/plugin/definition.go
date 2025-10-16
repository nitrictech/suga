package plugin

type GoPlugin struct {
	Alias  string `json:"Alias"`
	Name   string `json:"Name"`
	Import string `json:"Import"`
}

type PluginDefinition struct {
	Gets       []string   `json:"Gets"`
	Goproxies  []string   `json:"Goproxies,omitempty"` // List of GOPROXY URLs to use
	Pubsub     []GoPlugin `json:"Pubsub"`
	Storage    []GoPlugin `json:"Storage"`
	Service    GoPlugin   `json:"Service"`
}
