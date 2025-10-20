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

//go:embed kotlin_client_template
var kotlinClientTemplate string

// Kotlin package name validation (same as Java)
var kotlinPackageRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)

// Common Kotlin reserved keywords that cannot be used as package segments
var kotlinReservedWords = map[string]bool{
	// Java keywords (Kotlin is compatible)
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
	// Kotlin-specific keywords
	"as": true, "fun": true, "in": true, "is": true, "object": true, "typealias": true,
	"val": true, "var": true, "when": true, "by": true, "constructor": true,
	"delegate": true, "dynamic": true, "field": true, "file": true, "get": true,
	"init": true, "param": true, "property": true, "receiver": true, "set": true,
	"setparam": true, "where": true, "actual": true, "annotation": true, "companion": true,
	"crossinline": true, "data": true, "expect": true, "external": true,
	"infix": true, "inline": true, "inner": true, "internal": true, "lateinit": true,
	"noinline": true, "open": true, "operator": true, "out": true, "override": true,
	"reified": true, "sealed": true, "suspend": true, "tailrec": true, "vararg": true,
}

func validateKotlinPackageName(packageName string) error {
	if packageName == "" {
		return fmt.Errorf("Kotlin package name cannot be empty")
	}

	// Check overall pattern
	if !kotlinPackageRegex.MatchString(packageName) {
		return fmt.Errorf("invalid Kotlin package name '%s': must be dot-separated identifiers, each starting with a letter or underscore and followed only by letters, digits or underscores", packageName)
	}

	// Check each segment for reserved words
	segments := strings.Split(packageName, ".")
	for _, segment := range segments {
		if kotlinReservedWords[segment] {
			return fmt.Errorf("invalid Kotlin package name '%s': segment '%s' is a Kotlin reserved keyword", packageName, segment)
		}
	}

	return nil
}

type KotlinSDKTemplateData struct {
	Package string
	Buckets []ResourceNameNormalizer
}

func AppSpecToKotlinTemplateData(appSpec schema.Application, kotlinPackageName string) (KotlinSDKTemplateData, error) {
	buckets := []ResourceNameNormalizer{}
	for name, resource := range appSpec.GetResourceIntents() {
		if resource.GetType() != "bucket" {
			continue
		}

		normalized, err := NewResourceNameNormalizer(name)
		if err != nil {
			return KotlinSDKTemplateData{}, fmt.Errorf("failed to normalize resource name: %w", err)
		}

		buckets = append(buckets, normalized)
	}

	// Sort buckets deterministically to ensure stable generated code across runs
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Unmodified() < buckets[j].Unmodified()
	})

	return KotlinSDKTemplateData{
		Package: kotlinPackageName,
		Buckets: buckets,
	}, nil
}

// GenerateKotlin generates Kotlin SDK (backward compatible for Java users)
func GenerateKotlin(fs afero.Fs, appSpec schema.Application, outputDir string, kotlinPackageName string) error {
	if outputDir == "" {
		outputDir = fmt.Sprintf("%s/kotlin/client", version.CommandName)
	}

	if kotlinPackageName == "" {
		kotlinPackageName = "com.addsuga.client"
	}

	// Validate Kotlin package name
	if err := validateKotlinPackageName(kotlinPackageName); err != nil {
		return fmt.Errorf("invalid Kotlin package name: %w", err)
	}

	tmpl := template.Must(template.New("client").Parse(kotlinClientTemplate))
	data, err := AppSpecToKotlinTemplateData(appSpec, kotlinPackageName)
	if err != nil {
		return fmt.Errorf("failed to convert %s application spec into Kotlin SDK template data: %w", version.ProductName, err)
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

	filePath := filepath.Join(outputDir, "GeneratedSugaClient.kt")
	err = afero.WriteFile(fs, filePath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write generated file: %w", err)
	}

	fmt.Printf("Kotlin SDK generated at %s\n", filePath)

	return nil
}

// GenerateJava is kept for backward compatibility, but now generates Kotlin code
func GenerateJava(fs afero.Fs, appSpec schema.Application, outputDir string, packageName string) error {
	// For backward compatibility, redirect to Kotlin generation
	// but use java output directory if specified
	if outputDir != "" && strings.Contains(outputDir, "/java/") {
		outputDir = strings.Replace(outputDir, "/java/", "/kotlin/", 1)
	}
	return GenerateKotlin(fs, appSpec, outputDir, packageName)
}
