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

//go:embed java_client_template
var javaClientTemplate string

type JavaSDKTemplateData struct {
	Package string
	Buckets []ResourceNameNormalizer
}

func AppSpecToJavaTemplateData(appSpec schema.Application, javaPackageName string) (JavaSDKTemplateData, error) {
	buckets := []ResourceNameNormalizer{}
	for name, resource := range appSpec.GetResourceIntents() {
		if resource.GetType() != "bucket" {
			continue
		}

		normalized, err := NewResourceNameNormalizer(name)
		if err != nil {
			return JavaSDKTemplateData{}, fmt.Errorf("failed to normalize resource name: %w", err)
		}

		buckets = append(buckets, normalized)
	}

	return JavaSDKTemplateData{
		Package: javaPackageName,
		Buckets: buckets,
	}, nil
}

// GenerateJava generates Java SDK
func GenerateJava(fs afero.Fs, appSpec schema.Application, outputDir string, javaPackageName string) error {
	if outputDir == "" {
		outputDir = fmt.Sprintf("%s/java/client", version.CommandName)
	}

	if javaPackageName == "" {
		javaPackageName = "com.addsuga.client"
	}

	tmpl := template.Must(template.New("client").Parse(javaClientTemplate))
	data, err := AppSpecToJavaTemplateData(appSpec, javaPackageName)
	if err != nil {
		return fmt.Errorf("failed to convert %s application spec into Java SDK template data: %w", version.ProductName, err)
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

	filePath := filepath.Join(outputDir, "GeneratedSugaClient.java")
	err = afero.WriteFile(fs, filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write generated file: %w", err)
	}

	fmt.Printf("Java SDK generated at %s\n", filePath)

	return nil
}
