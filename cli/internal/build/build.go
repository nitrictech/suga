package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/cli/internal/plugins"
	"github.com/nitrictech/suga/cli/internal/pluginserver"
	"github.com/nitrictech/suga/cli/pkg/schema"
	"github.com/nitrictech/suga/engines/terraform"
	"github.com/samber/do/v2"
	"github.com/spf13/afero"
)

type BuilderService struct {
	fs        afero.Fs
	apiClient *api.SugaApiClient
	logDir    string // Optional directory for panic logs
}

// sanitizeForFilename removes or replaces characters that aren't safe for filenames
func sanitizeForFilename(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

	return re.ReplaceAllString(input, "_")
}

type BuildOptions struct {
	LibraryReplacements map[string]string
}

func (b *BuilderService) BuildProject(appSpec *schema.Application, currentTeam string, opts BuildOptions) (string, error) {
	if appSpec.Target == "" {
		return "", fmt.Errorf("no target specified in project %s", appSpec.Name)
	}

	onDemandPluginRepo := plugins.NewPluginRepository(b.apiClient, currentTeam)

	var platformRepo terraform.PlatformRepository = nil
	if !strings.HasPrefix(appSpec.Target, terraform.PlatformReferencePrefix_File) {
		repo, err := NewRepository(b.apiClient, currentTeam, appSpec.Target)
		if err != nil {
			return "", err
		}
		platformRepo = repo
	}

	platform, err := terraform.PlatformFromId(b.fs, appSpec.Target, platformRepo)
	if err != nil {
		return "", err
	}

	for library, target := range opts.LibraryReplacements {
		if err := platform.ReplaceLibrary(library, target); err != nil {
			return "", fmt.Errorf("failed to replace library %s: %w", library, err)
		}
		fmt.Printf("Replacing: %s -> %s\n", library, target)
	}

	compositeRepo := pluginserver.NewCompositePluginRepository(platform, onDemandPluginRepo)

	engine := terraform.New(platform, terraform.WithRepository(compositeRepo))

	stackPath, err := engine.Apply(appSpec)
	if err != nil {
		return "", b.processBuildError(err, appSpec.Target)
	}
	return stackPath, nil
}

func (b *BuilderService) BuildProjectFromFile(projectFile, currentTeam string, opts BuildOptions) (string, error) {
	appSpec, err := schema.LoadFromFile(b.fs, projectFile, true)
	if err != nil {
		return "", fmt.Errorf("failed to load project file: %w", err)
	}

	return b.BuildProject(appSpec, currentTeam, opts)
}

func NewBuilderService(injector do.Injector) (*BuilderService, error) {
	fs := do.MustInvoke[afero.Fs](injector)
	apiClient := do.MustInvoke[*api.SugaApiClient](injector)

	return &BuilderService{
		fs:        fs,
		apiClient: apiClient,
		logDir:    "", // No logging by default
	}, nil
}

// WithPanicLogging configures the builder to log panics to the specified directory
func (b *BuilderService) WithPanicLogging(logDir string) *BuilderService {
	b.logDir = logDir
	return b
}

// processBuildError handles error formatting and optional logging
func (b *BuilderService) processBuildError(err error, target string) error {
	var panicErr *terraform.PanicError
	if errors.As(err, &panicErr) {
		if b.logDir != "" {
			if logFilePath, logErr := b.logPanicToFile(panicErr, target); logErr == nil {
				return fmt.Errorf("a panic occurred building with %s. panic details logged to: %s", target, logFilePath)
			}
		}
		return fmt.Errorf("a panic occurred building with %s. panic: %v", target, panicErr.OriginalPanic)
	}

	return fmt.Errorf("an error occurred building with %s. error: %v", target, err)
}

// logPanicToFile logs panic details to the configured log directory
func (b *BuilderService) logPanicToFile(panicErr *terraform.PanicError, target string) (string, error) {
	err := os.MkdirAll(b.logDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create unique filename with timestamp and sanitized target name
	timestamp := time.Now().Format("20060102-150405")
	sanitizedTarget := sanitizeForFilename(target)
	logFileName := fmt.Sprintf("panic-%s-%s.log", sanitizedTarget, timestamp)
	logFilePath := filepath.Join(b.logDir, logFileName)

	// Create detailed log content
	logContent := "Suga Build Panic Report\n"
	logContent += "========================\n"
	logContent += fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
	logContent += fmt.Sprintf("Target: %s\n", target)
	logContent += fmt.Sprintf("Panic: %v\n\n", panicErr.OriginalPanic)
	logContent += fmt.Sprintf("Stack Trace:\n%s\n", panicErr.StackTrace)

	// Write to log file
	err = os.WriteFile(logFilePath, []byte(logContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write log file: %v", err)
	}

	return logFilePath, nil
}
