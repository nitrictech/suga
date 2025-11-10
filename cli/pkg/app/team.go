package app

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/cli/internal/config"
	"github.com/nitrictech/suga/cli/internal/style"
	"github.com/nitrictech/suga/cli/internal/style/colors"
	"github.com/nitrictech/suga/cli/internal/style/icons"
	"github.com/nitrictech/suga/cli/internal/version"
	"github.com/nitrictech/suga/cli/internal/workos"
	"github.com/nitrictech/suga/cli/pkg/tui"
	"github.com/nitrictech/suga/cli/pkg/tui/ask"
	"github.com/samber/do/v2"
)

type TeamApp struct {
	config    *config.Config
	apiClient *api.SugaApiClient
	// auth uses WorkOSAuth directly because team switching requires
	// WorkOS-specific organization ID during token refresh
	auth   *workos.WorkOSAuth
	styles tui.AppStyles
}

func NewTeamApp(injector do.Injector) (*TeamApp, error) {
	config := do.MustInvoke[*config.Config](injector)
	apiClient, err := api.NewSugaApiClient(injector)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Use WorkOSAuth directly for team switching functionality
	auth := do.MustInvoke[*workos.WorkOSAuth](injector)

	styles := tui.NewAppStyles()

	return &TeamApp{
		config:    config,
		apiClient: apiClient,
		auth:      auth,
		styles:    styles,
	}, nil
}

func (t *TeamApp) SwitchTeam(teamSlug string) error {
	allTeams, err := t.apiClient.GetUserTeams()
	if err != nil {
		if errors.Is(err, api.ErrUnauthenticated) {
			fmt.Println("Please login first, using the", t.styles.Emphasize.Render(version.GetCommand("login")), "command")
			return nil
		}
		fmt.Printf("Failed to get teams: %v\n", err)
		return nil
	}

	if len(allTeams) == 0 {
		url := "the Suga dashboard"
		if t.config != nil && strings.TrimSpace(t.config.Url) != "" {
			url = t.config.Url
		}
		fmt.Printf("No teams found. Navigate to %s in your browser to create your first team.\n", style.Teal(url))
		return nil
	}

	if teamSlug != "" {
		return t.switchToTeamBySlug(allTeams, teamSlug)
	}

	return t.showInteractiveTeamPicker(allTeams)
}

func (t *TeamApp) switchToTeamBySlug(teams []api.Team, slug string) error {
	var targetTeam *api.Team
	for _, team := range teams {
		if team.Slug == slug {
			targetTeam = &team
			break
		}
	}

	if targetTeam == nil {
		return fmt.Errorf("team not found: %s", slug)
	}

	if targetTeam.IsCurrent {
		fmt.Printf("%s Already using team: %s\n", style.Gray(icons.Check), style.Teal(targetTeam.Name))
		return nil
	}

	return t.performTeamSwitch(targetTeam)
}

func (t *TeamApp) showInteractiveTeamPicker(teams []api.Team) error {
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].Name < teams[j].Name
	})

	var currentTeam *api.Team
	for i := range teams {
		if teams[i].IsCurrent {
			currentTeam = &teams[i]
			break
		}
	}

	if currentTeam != nil {
		currentStyle := lipgloss.NewStyle().
			Foreground(colors.Teal).
			Bold(true)
		fmt.Printf("Current team: %s\n\n", currentStyle.Render(currentTeam.Name))
	}

	teamMap := make(map[string]*api.Team)
	optionLabels := make([]string, 0, len(teams))

	for i := range teams {
		team := &teams[i]
		label := team.Name
		if team.IsCurrent {
			label = fmt.Sprintf("%s (current)", team.Name)
		}
		teamMap[label] = team
		optionLabels = append(optionLabels, label)
	}

	var selectedLabel string
	err := ask.NewSelect[string]().
		Title("Select a team:").
		Options(huh.NewOptions(optionLabels...)...).
		Value(&selectedLabel).
		Height(len(teams) + 2). // +2 for extra spacing at the bottom
		Run()

	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return fmt.Errorf("failed to get team selection: %w", err)
	}

	selectedTeam := teamMap[selectedLabel]
	if selectedTeam.IsCurrent {
		fmt.Printf("%s Already using team: %s\n", style.Gray(icons.Check), style.Teal(selectedTeam.Name))
		return nil
	}

	return t.performTeamSwitch(selectedTeam)
}

func (t *TeamApp) performTeamSwitch(team *api.Team) error {
	fmt.Printf("Switching to team: %s\n", style.Teal(team.Name))

	err := t.auth.RefreshToken(workos.RefreshTokenOptions{
		OrganizationID: team.WorkOsID,
	})
	if err != nil {
		fmt.Printf("%s Failed to refresh token for organization: %v\n", style.Red(icons.Cross), err)
		fmt.Printf("Try running %s to re-authenticate\n", style.Teal(version.CommandName+" login"))
		return nil
	}

	fmt.Printf("%s Successfully switched to team: %s\n", style.Green(icons.Check), style.Teal(team.Name))
	return nil
}
