package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"text/template"

	"github.com/nitrictech/suga/server/plugin"
)

//go:embed main.tmpl
var mainTmpl string

// fetchModulesFromProxy fetches the list of discovered modules from a proxy server
func fetchModulesFromProxy(proxyURL string) ([]string, error) {
	resp, err := http.Get(proxyURL + "/api/modules")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch modules from %s: %w", proxyURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch modules from %s: status %d", proxyURL, resp.StatusCode)
	}

	var result struct {
		Modules []string `json:"modules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode modules response: %w", err)
	}

	return result.Modules, nil
}

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

		// Fetch discovered modules from each proxy and build GONOSUMDB
		var modules []string
		for _, proxy := range pluginDef.Goproxies {
			proxyModules, err := fetchModulesFromProxy(proxy)
			if err != nil {
				continue
			}
			for _, module := range proxyModules {
				if !slices.Contains(modules, module) {
					modules = append(modules, module)
				}
			}
		}

		// Set GONOSUMDB for discovered modules
		if len(modules) > 0 {
			env = append(env, fmt.Sprintf("GONOSUMDB=%s", strings.Join(modules, ",")))
		}
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
