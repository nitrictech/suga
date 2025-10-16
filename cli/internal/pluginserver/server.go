package pluginserver

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type PluginServer struct {
	fs          afero.Fs
	mux         *http.ServeMux
	basePath    string                 // Base directory to search for plugins and modules
	pluginCache map[string]*PluginInfo // Cache of discovered plugins by name
	moduleCache []string               // Cache of discovered Go module names
}

func NewPluginServer(fs afero.Fs, basePath string) (*PluginServer, error) {
	plugins, err := discoverPlugins(fs, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover plugins: %w", err)
	}

	modules, err := discoverModules(fs, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover modules: %w", err)
	}

	s := &PluginServer{
		fs:          fs,
		mux:         http.NewServeMux(),
		basePath:    basePath,
		pluginCache: plugins,
		moduleCache: modules,
	}

	s.setupRoutes()

	return s, nil
}

func (s *PluginServer) setupRoutes() {
	// Plugin manifest endpoints (authenticated)
	s.mux.HandleFunc("/api/teams/{team}/plugin_libraries/{lib}/versions/{version}/plugins/{name}",
		s.handleGetPluginManifest)

	// Plugin manifest endpoints (public)
	s.mux.HandleFunc("/api/public/plugin_libraries/{team}/{lib}/versions/{version}/plugins/{name}",
		s.handleGetPluginManifest)

	// Terraform module zip endpoint
	s.mux.HandleFunc("/terraform-modules/", s.handleTerraformModuleZip)

	// Go modules discovery endpoint
	s.mux.HandleFunc("/api/modules", s.handleDiscoverModules)

	// Go module proxy endpoints - use catch-all pattern
	s.mux.HandleFunc("/", s.handleModuleProxy)
}

func (s *PluginServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *PluginServer) handleGetPluginManifest(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	// Search for manifest file by walking plugin paths and checking name field
	_, manifestData, err := findPluginManifest(s.fs, s.basePath, name)
	if err != nil {
		http.Error(w, fmt.Sprintf("plugin %s not found", name), http.StatusNotFound)
		return
	}

	// Parse manifest (always YAML)
	var manifest map[string]interface{}
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse manifest: %v", err), http.StatusInternalServerError)
		return
	}

	// Return manifest in API format
	response := api.GetPluginManifestResponse{
		Manifest: manifest,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleTerraformModuleZip serves Terraform modules as zip archives
// Expected URL format: /terraform-modules/{plugin-name}.zip
func (s *PluginServer) handleTerraformModuleZip(w http.ResponseWriter, r *http.Request) {
	// Extract plugin name from URL (remove /terraform-modules/ prefix and .zip suffix)
	pluginName := strings.TrimPrefix(r.URL.Path, "/terraform-modules/")
	pluginName = strings.TrimSuffix(pluginName, ".zip")

	if pluginName == "" {
		http.Error(w, "plugin name is required", http.StatusBadRequest)
		return
	}

	// Look up plugin in cache
	pluginInfo, ok := s.pluginCache[pluginName]
	if !ok {
		http.Error(w, fmt.Sprintf("plugin %s not found", pluginName), http.StatusNotFound)
		return
	}

	if pluginInfo.TerraformModulePath == "" {
		http.Error(w, fmt.Sprintf("plugin %s has no terraform module", pluginName), http.StatusBadRequest)
		return
	}

	// Resolve the terraform module path relative to the plugin directory
	moduleDir := filepath.Join(pluginInfo.Dir, pluginInfo.TerraformModulePath)

	// Check if the resolved path exists
	if exists, _ := afero.Exists(s.fs, moduleDir); !exists {
		http.Error(w, fmt.Sprintf("terraform module not found at %s", pluginInfo.TerraformModulePath), http.StatusNotFound)
		return
	}

	// Create zip archive of the module
	w.Header().Set("Content-Type", "application/zip")
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Walk the module directory and add files to zip
	err := afero.Walk(s.fs, moduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(moduleDir, path)
		if err != nil {
			return err
		}

		// Create zip entry
		zipFile, err := zipWriter.Create(filepath.ToSlash(relPath))
		if err != nil {
			return err
		}

		// Copy file contents
		data, err := afero.ReadFile(s.fs, path)
		if err != nil {
			return err
		}

		_, err = zipFile.Write(data)
		return err
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create zip: %v", err), http.StatusInternalServerError)
		return
	}
}

// findPluginManifest searches for a plugin manifest by name field
// Returns the manifest path and content, or an error if not found
func findPluginManifest(fs afero.Fs, basePath, pluginName string) (string, []byte, error) {
	var foundPath string
	var foundData []byte

	// Walk through the base path looking for manifests
	err := afero.Walk(fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if this is a manifest file (yaml or yml only)
		filename := filepath.Base(path)
		if filename != "manifest.yaml" && filename != "manifest.yml" {
			return nil
		}

		// Read and parse manifest
		manifestData, err := afero.ReadFile(fs, path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var manifest map[string]interface{}
		if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
			return nil // Skip invalid manifests
		}

		// Check if this manifest matches the requested plugin name
		if name, ok := manifest["name"].(string); ok && name == pluginName {
			foundPath = path
			foundData = manifestData
			return filepath.SkipAll // Stop walking once found
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", nil, err
	}

	if foundPath != "" {
		return foundPath, foundData, nil
	}

	return "", nil, fmt.Errorf("plugin %s not found", pluginName)
}

func (s *PluginServer) Start(addr string) error {
	return http.ListenAndServe(addr, s)
}

// Go module proxy handlers
// Implements the Go module proxy protocol: https://golang.org/ref/mod#goproxy-protocol

// handleModuleProxy routes module proxy requests to the appropriate handler
func (s *PluginServer) handleModuleProxy(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check if this is a module proxy request
	if strings.Contains(path, "/@v/") {
		// Extract module path and check the endpoint
		parts := strings.Split(path, "/@v/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}

		modulePath := strings.TrimPrefix(parts[0], "/")
		endpoint := parts[1]

		// Route to appropriate handler based on endpoint
		if endpoint == "list" {
			s.handleModuleListWithPath(w, r, modulePath)
		} else if strings.HasSuffix(endpoint, ".info") {
			version := strings.TrimSuffix(endpoint, ".info")
			s.handleModuleInfoWithPath(w, r, modulePath, version)
		} else if strings.HasSuffix(endpoint, ".mod") {
			version := strings.TrimSuffix(endpoint, ".mod")
			s.handleModuleModWithPath(w, r, modulePath, version)
		} else if strings.HasSuffix(endpoint, ".zip") {
			version := strings.TrimSuffix(endpoint, ".zip")
			s.handleModuleZipWithPath(w, r, modulePath, version)
		} else {
			http.NotFound(w, r)
		}
		return
	}

	// Check if this is a @latest request
	if strings.HasSuffix(path, "/@latest") {
		modulePath := strings.TrimPrefix(strings.TrimSuffix(path, "/@latest"), "/")
		s.handleModuleLatestWithPath(w, r, modulePath)
		return
	}

	// Not a module proxy request
	http.NotFound(w, r)
}

func (s *PluginServer) handleModuleListWithPath(w http.ResponseWriter, r *http.Request, modulePath string) {
	// Find the module in our module paths
	_, version, err := s.findModule(modulePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return single version (the one we found)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%s\n", version)
}

func (s *PluginServer) handleModuleInfoWithPath(w http.ResponseWriter, r *http.Request, modulePath, version string) {
	// Verify module exists
	moduleDir, _, err := s.findModule(modulePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return module info
	info := struct {
		Version string    `json:"Version"`
		Time    time.Time `json:"Time"`
	}{
		Version: version,
		Time:    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)

	_ = moduleDir // Use variable
}

func (s *PluginServer) handleModuleModWithPath(w http.ResponseWriter, r *http.Request, modulePath, version string) {
	// Find the module
	moduleDir, _, err := s.findModule(modulePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Serve the go.mod file
	goModPath := filepath.Join(moduleDir, "go.mod")
	data, err := afero.ReadFile(s.fs, goModPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("go.mod not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(data)
}

func (s *PluginServer) handleModuleZipWithPath(w http.ResponseWriter, r *http.Request, modulePath, version string) {
	// Find the module
	moduleDir, _, err := s.findModule(modulePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Create zip archive of the module
	w.Header().Set("Content-Type", "application/zip")
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Walk the module directory and add files to zip
	err = afero.Walk(s.fs, moduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(moduleDir, path)
		if err != nil {
			return err
		}

		// Add module path prefix as required by Go module proxy
		zipPath := filepath.Join(fmt.Sprintf("%s@%s", modulePath, version), relPath)

		// Create zip entry
		zipFile, err := zipWriter.Create(filepath.ToSlash(zipPath))
		if err != nil {
			return err
		}

		// Copy file contents
		data, err := afero.ReadFile(s.fs, path)
		if err != nil {
			return err
		}

		_, err = zipFile.Write(data)
		return err
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create zip: %v", err), http.StatusInternalServerError)
		return
	}
}

func (s *PluginServer) handleModuleLatestWithPath(w http.ResponseWriter, r *http.Request, modulePath string) {
	// Find the module
	_, version, err := s.findModule(modulePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return latest version info
	info := struct {
		Version string    `json:"Version"`
		Time    time.Time `json:"Time"`
	}{
		Version: version,
		Time:    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

// findModule searches for a Go module in the base path
// Returns the module directory and version
// For package paths like github.com/foo/bar/pkg, it finds the module root by looking for go.mod
func (s *PluginServer) findModule(modulePath string) (string, string, error) {
	moduleDir, err := findModuleInPath(s.fs, s.basePath, modulePath)
	if err == nil {
		return moduleDir, "v0.0.0-dev", nil
	}

	return "", "", fmt.Errorf("module %s not found in local paths", modulePath)
}

// findModuleInPath searches for a go.mod file that matches the module path
func findModuleInPath(fs afero.Fs, basePath, modulePath string) (string, error) {
	var foundModuleDir string

	// Walk the base path looking for go.mod files
	err := afero.Walk(fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking on errors
		}

		// Look for go.mod files
		if !info.IsDir() && info.Name() == "go.mod" {
			// Read go.mod to check if it matches our module
			data, err := afero.ReadFile(fs, path)
			if err != nil {
				return nil
			}

			// Parse first line: "module github.com/foo/bar"
			lines := strings.Split(string(data), "\n")
			if len(lines) > 0 {
				firstLine := strings.TrimSpace(lines[0])
				if strings.HasPrefix(firstLine, "module ") {
					moduleDecl := strings.TrimPrefix(firstLine, "module ")
					moduleDecl = strings.TrimSpace(moduleDecl)

					// Check if the requested module path starts with this module
					if modulePath == moduleDecl || strings.HasPrefix(modulePath, moduleDecl+"/") {
						foundModuleDir = filepath.Dir(path)
						return filepath.SkipAll // Stop walking
					}
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", err
	}

	if foundModuleDir != "" {
		return foundModuleDir, nil
	}

	return "", fmt.Errorf("module not found")
}

// discoverPlugins scans the base path and returns discovered plugins as a map
func discoverPlugins(fs afero.Fs, basePath string) (map[string]*PluginInfo, error) {
	plugins := make(map[string]*PluginInfo)

	// Walk through looking for manifest.yaml or manifest.yml files
	err := afero.Walk(fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if this is a manifest file (yaml or yml only)
		filename := filepath.Base(path)
		if filename != "manifest.yaml" && filename != "manifest.yml" {
			return nil
		}

		// Read and parse manifest to get the plugin name
		manifestData, err := afero.ReadFile(fs, path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var manifest map[string]interface{}
		if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
			return nil // Skip invalid manifests
		}

		// Extract name from manifest
		pluginName, ok := manifest["name"].(string)
		if !ok || pluginName == "" {
			return nil // Skip if no name in manifest
		}

		// Extract terraform module path from deployment section
		var terraformModulePath string
		if deployment, ok := manifest["deployment"].(map[string]interface{}); ok {
			if tfPath, ok := deployment["terraform"].(string); ok {
				terraformModulePath = tfPath
			}
		}

		pluginInfo := PluginInfo{
			Name:                pluginName,
			Path:                path,
			Dir:                 filepath.Dir(path),
			TerraformModulePath: terraformModulePath,
		}

		plugins[pluginName] = &pluginInfo

		return nil
	})

	if err != nil {
		return nil, err
	}

	return plugins, nil
}

// discoverModules scans the base path and returns discovered Go modules
func discoverModules(fs afero.Fs, basePath string) ([]string, error) {
	modules := []string{}
	visited := make(map[string]bool)

	err := afero.Walk(fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking on errors
		}

		// Look for go.mod files
		if !info.IsDir() && info.Name() == "go.mod" {
			// Read go.mod to get the module name
			data, err := afero.ReadFile(fs, path)
			if err != nil {
				return nil
			}

			// Parse first line: "module github.com/foo/bar"
			lines := strings.Split(string(data), "\n")
			if len(lines) > 0 {
				firstLine := strings.TrimSpace(lines[0])
				if strings.HasPrefix(firstLine, "module ") {
					moduleDecl := strings.TrimPrefix(firstLine, "module ")
					moduleDecl = strings.TrimSpace(moduleDecl)

					// Add to list if not already seen
					if !visited[moduleDecl] {
						modules = append(modules, moduleDecl)
						visited[moduleDecl] = true
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return modules, nil
}

// handleDiscoverModules returns the cached list of discovered Go modules
func (s *PluginServer) handleDiscoverModules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"modules": s.moduleCache,
	})
}

// GetPlugins returns the cached list of discovered plugins
func (s *PluginServer) GetPlugins() []PluginInfo {
	plugins := []PluginInfo{}
	for _, plugin := range s.pluginCache {
		plugins = append(plugins, *plugin)
	}
	return plugins
}

// GetModules returns the cached list of discovered Go modules
func (s *PluginServer) GetModules() []string {
	return s.moduleCache
}

type PluginInfo struct {
	Name                string
	Path                string // Path to the manifest file
	Dir                 string // Directory containing the manifest
	TerraformModulePath string // Relative path to the terraform module from manifest
}

func (p PluginInfo) String() string {
	return p.Name
}
