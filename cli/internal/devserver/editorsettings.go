package devserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nitrictech/suga/cli/internal/config"
	"github.com/nitrictech/suga/cli/internal/version"
)

type EditorSettingsSync struct{
	fileSync *SugaFileSync
}

type EditorSettings struct {
	SelectedTarget string `json:"selectedTarget,omitempty"`
}

type EditorSettingsUpdate Message[EditorSettings]

func NewEditorSettingsSync(fileSync *SugaFileSync) *EditorSettingsSync {
	return &EditorSettingsSync{
		fileSync: fileSync,
	}
}

func (ess *EditorSettingsSync) OnConnect(send SendFunc) {
	// Load existing editor settings, use empty settings if load fails
	settings, err := loadEditorSettings()
	if err != nil {
		fmt.Println("Could not load editor settings:", err)
		settings = EditorSettings{}
	}

	// Validate the selected target against current application targets
	if err == nil && settings.SelectedTarget != "" && ess.fileSync != nil {
		application, _, appErr := ess.fileSync.getApplicationFileContents()
		if appErr == nil && application != nil && !isValidTarget(settings.SelectedTarget, application.Targets) {
			fmt.Printf("Invalid target '%s' found in editor settings, clearing\n", settings.SelectedTarget)
			settings.SelectedTarget = ""
			// Store the corrected settings
			if err := storeEditorSettings(settings); err != nil {
				fmt.Println("Error storing corrected editor settings:", err)
			}
		}
	}

	send(Message[any]{
		Type:    "editorSettingsMessage",
		Payload: settings,
	})
}

func (ess *EditorSettingsSync) OnMessage(message json.RawMessage) {
	var editorSettingsUpdate EditorSettingsUpdate

	err := json.Unmarshal(message, &editorSettingsUpdate)
	if err != nil {
		fmt.Printf("Error parsing editor settings message: %v\n", err)
		return
	}

	if editorSettingsUpdate.Type != "editorSettingsMessage" {
		return
	}

	err = storeEditorSettings(editorSettingsUpdate.Payload)
	if err != nil {
		fmt.Println("Error storing editor settings:", err)
	}
}

func loadEditorSettings() (EditorSettings, error) {
	editorSettingsPath := filepath.Join(config.LocalConfigPath(), "editor-settings.json")

	data, err := os.ReadFile(editorSettingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty settings
			return EditorSettings{}, nil
		}
		return EditorSettings{}, fmt.Errorf("failed to read editor settings file: %w", err)
	}

	var settings EditorSettings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return EditorSettings{}, fmt.Errorf("failed to unmarshal editor settings: %w", err)
	}

	return settings, nil
}

func storeEditorSettings(newSettings EditorSettings) error {
	if err := os.MkdirAll(config.LocalConfigPath(), 0755); err != nil {
		return fmt.Errorf("failed to create %s config directory: %w", version.CommandName, err)
	}

	existingSettings, err := loadEditorSettings()
	if err != nil {
		// If we can't load existing settings, start with empty settings
		existingSettings = EditorSettings{}
	}

	// Merge new settings with existing ones
	// Always update SelectedTarget, including empty string (which clears the selection)
	existingSettings.SelectedTarget = newSettings.SelectedTarget

	data, err := json.MarshalIndent(existingSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal editor settings: %w", err)
	}

	editorSettingsPath := filepath.Join(config.LocalConfigPath(), "editor-settings.json")
	err = os.WriteFile(editorSettingsPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write editor settings file: %w", err)
	}

	return nil
}

func isValidTarget(target string, validTargets []string) bool {
	for _, validTarget := range validTargets {
		if target == validTarget {
			return true
		}
	}
	return false
}