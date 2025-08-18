package ask

import (
	"github.com/charmbracelet/huh"
	"github.com/nitrictech/suga/cli/internal/style/colors"
)

func NewInput() *huh.Input {
	return huh.NewInput().
		Inline(true).
		Prompt(" ").
		WithTheme(&colors.Theme).(*huh.Input)
}
