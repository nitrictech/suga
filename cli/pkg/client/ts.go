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

//go:embed ts_client_template
var tsClientTemplate string

type TSSDKTemplateData struct {
	Package string
	Buckets []BucketWithPermissions
}

func AppSpecToTSTemplateData(appSpec schema.Application) (TSSDKTemplateData, error) {
	buckets, err := ExtractPermissionsForBuckets(appSpec)
	if err != nil {
		return TSSDKTemplateData{}, err
	}

	return TSSDKTemplateData{
		Package: "client",
		Buckets: buckets,
	}, nil
}

// GenerateTypeScript generates TypeScript SDK
func GenerateTypeScript(fs afero.Fs, appSpec schema.Application, outputDir string) error {
	if outputDir == "" {
		outputDir = fmt.Sprintf("%s/ts/client", version.CommandName)
	}

	tmpl := template.Must(template.New("client").Parse(tsClientTemplate))
	data, err := AppSpecToTSTemplateData(appSpec)
	if err != nil {
		return fmt.Errorf("failed to convert %s application spec into TypeScript SDK template data: %w", version.ProductName, err)
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

	filePath := filepath.Join(outputDir, "index.ts")
	err = afero.WriteFile(fs, filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write generated file: %w", err)
	}

	fmt.Printf("TypeScript SDK generated at %s\n", filePath)

	return nil
}
