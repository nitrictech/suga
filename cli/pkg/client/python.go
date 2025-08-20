package client

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	_ "embed"

	"github.com/nitrictech/suga/cli/internal/version"
	"github.com/nitrictech/suga/cli/pkg/schema"
	"github.com/spf13/afero"
)

//go:embed python_client_template
var pyClientTemplate string


type PySDKTemplateData struct {
	Package string
	Buckets []BucketWithPermissions
}

func AppSpecToPyTemplateData(appSpec schema.Application) (PySDKTemplateData, error) {
	buckets, err := ExtractPermissionsForBuckets(appSpec)
	if err != nil {
		return PySDKTemplateData{}, err
	}

	return PySDKTemplateData{
		Package: "client",
		Buckets: buckets,
	}, nil
}

func GeneratePython(fs afero.Fs, appSpec schema.Application, outputDir string) error {
	if outputDir == "" {
		// Add _gen suffix so the generated client doesn't shadow the 'suga' import from Pypi
		outputDir = fmt.Sprintf("%s_gen", version.CommandName)
	}

	tmpl := template.Must(template.New("client").Parse(pyClientTemplate))
	data, err := AppSpecToPyTemplateData(appSpec)
	if err != nil {
		return fmt.Errorf("failed to convert %s application spec into Python SDK template data: %w", version.ProductName, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	err = fs.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filePath := filepath.Join(outputDir, "client.py")
	err = afero.WriteFile(fs, filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write generated file: %w", err)
	}

	fmt.Printf("Python SDK generated at %s\n", filePath)

	return nil
}
