package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/nitrictech/suga/server/plugin"
)

//go:embed main.tmpl
var mainTmpl string

func main() {
	// template our main.go by injecting the plugin name and known plugin constructor
	tmpl, err := template.New("main").Parse(mainTmpl)
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}

	pluginDefEnv := os.Getenv("PLUGIN_DEFINITION")
	if pluginDefEnv == "" {
		log.Fatalf("PLUGIN_DEFINITION is not set")
	}

	var pluginDef plugin.PluginDefinition
	if err := json.Unmarshal([]byte(pluginDefEnv), &pluginDef); err != nil {
		log.Fatalf("error unmarshaling plugin definition: %v", err)
	}

	// Set GOPROXY if we have any custom proxies
	env := os.Environ()
	if len(pluginDef.Goproxies) > 0 {
		// Join proxies with comma and append public proxy before direct as fallback
		// This ensures we use the public Go proxy (with proper checksums) before falling back to direct downloads
		goproxy := strings.Join(pluginDef.Goproxies, ",") + ",https://proxy.golang.org,direct"
		env = append(env, "GOPROXY="+goproxy)

		// Disable checksum validation during local development
		// TODO: Limit this to modules served from the proxies only
		env = append(env, "GONOSUMDB=*")
	}

	for _, get := range pluginDef.Gets {
		cmd := exec.Command("go", "get", "-u", get)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			log.Fatalf("error running go get: %v", err)
		}
	}

	err = tmpl.Execute(os.Stdout, pluginDef)
	if err != nil {
		log.Fatalf("error executing template: %v", err)
	}
}
