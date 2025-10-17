package client

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	_ "embed"

	"github.com/nitrictech/suga/cli/internal/version"
	"github.com/nitrictech/suga/cli/pkg/schema"
	"github.com/spf13/afero"
)

//go:embed java_client_template
var javaClientTemplate string

// Java package name validation
var javaPackageRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)

// Common Java reserved keywords that cannot be used as package segments
var javaReservedWords = map[string]bool{
	"abstract": true, "assert": true, "boolean": true, "break": true, "byte": true,
	"case": true, "catch": true, "char": true, "class": true, "const": true,
	"continue": true, "default": true, "do": true, "double": true, "else": true,
	"enum": true, "extends": true, "final": true, "finally": true, "float": true,
	"for": true, "goto": true, "if": true, "implements": true, "import": true,
	"instanceof": true, "int": true, "interface": true, "long": true, "native": true,
	"new": true, "null": true, "package": true, "private": true, "protected": true,
	"public": true, "return": true, "short": true, "static": true, "strictfp": true,
	"super": true, "switch": true, "synchronized": true, "this": true, "throw": true,
	"throws": true, "transient": true, "try": true, "void": true, "volatile": true,
	"while": true, "true": true, "false": true,
}

func validateJavaPackageName(packageName string) error {
	if packageName == "" {
		return fmt.Errorf("Java package name cannot be empty")
	}

	// Check overall pattern
	if !javaPackageRegex.MatchString(packageName) {
		return fmt.Errorf("invalid Java package name '%s': must be dot-separated identifiers, each starting with a letter or underscore and followed only by letters, digits or underscores", packageName)
	}

	// Check each segment for reserved words
	segments := strings.Split(packageName, ".")
	for _, segment := range segments {
		if javaReservedWords[segment] {
			return fmt.Errorf("invalid Java package name '%s': segment '%s' is a Java reserved keyword", packageName, segment)
		}
	}

	return nil
}

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

	// Sort buckets deterministically to ensure stable generated code across runs
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Unmodified() < buckets[j].Unmodified()
	})

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

	// Validate Java package name
	if err := validateJavaPackageName(javaPackageName); err != nil {
		return fmt.Errorf("invalid Java package name: %w", err)
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
