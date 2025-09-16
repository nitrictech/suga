package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// fetchTemplateList is a helper function to fetch and unmarshal template lists
func (c *SugaApiClient) fetchTemplateList(url string, errorContext string) ([]TemplateResponse, error) {
	response, err := c.get(url, true)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get %s: %s", errorContext, response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var templates ListTemplatesResponse
	if err := json.Unmarshal(body, &templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %v, body: %s", errorContext, err, string(body))
	}

	return templates.Templates, nil
}

// tryTemplateEndpoint attempts to fetch a template from an endpoint with fallback error handling
func (c *SugaApiClient) tryTemplateEndpoint(url string) (*TemplateVersion, error) {
	response, err := c.get(url, true)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", response.StatusCode, response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var templateResponse GetTemplateVersionResponse
	if err := json.Unmarshal(body, &templateResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %v, body: %s", err, string(body))
	}

	return templateResponse.Template, nil
}

func (c *SugaApiClient) GetTemplates(team string) ([]TemplateResponse, error) {
	// Fetch team templates and public templates in parallel
	type templateResult struct {
		templates []TemplateResponse
		err       error
	}

	teamChan := make(chan templateResult, 1)
	publicChan := make(chan templateResult, 1)

	// Fetch team templates
	go func() {
		teamURL := fmt.Sprintf("/api/teams/%s/templates", url.PathEscape(team))
		templates, err := c.fetchTemplateList(teamURL, "team templates")
		teamChan <- templateResult{templates, err}
	}()

	// Fetch public templates from configured teams
	go func() {
		var publicURL string
		if len(c.publicTemplatesTeams) == 0 {
			// No team filter - get all public templates
			publicURL = "/api/public/templates"
		} else {
			// Build query string with multiple team slugs
			params := url.Values{}
			for _, team := range c.publicTemplatesTeams {
				params.Add("team_slug", team)
			}
			publicURL = "/api/public/templates?" + params.Encode()
		}
		templates, err := c.fetchTemplateList(publicURL, "public templates")
		publicChan <- templateResult{templates, err}
	}()

	// Collect results
	teamResult := <-teamChan
	publicResult := <-publicChan

	// Handle errors - if team templates fail, we still want public templates
	var allTemplates []TemplateResponse
	if teamResult.err == nil {
		allTemplates = append(allTemplates, teamResult.templates...)
	}
	if publicResult.err == nil {
		allTemplates = append(allTemplates, publicResult.templates...)
	}

	// If both failed, return the team error (primary)
	if teamResult.err != nil && publicResult.err != nil {
		return nil, teamResult.err
	}

	// Deduplicate templates (prevent exact duplicates)
	seen := make(map[string]bool)
	var uniqueTemplates []TemplateResponse
	for _, template := range allTemplates {
		key := fmt.Sprintf("%s/%s", template.TeamSlug, template.Slug)
		if !seen[key] {
			seen[key] = true
			uniqueTemplates = append(uniqueTemplates, template)
		}
	}

	return uniqueTemplates, nil
}

// GetTemplate gets a specific template by teamSlug, templateName and version
// version is optional, if it is not provided, the latest version will be returned
func (c *SugaApiClient) GetTemplate(teamSlug string, templateName string, version string) (*TemplateVersion, error) {
	// Build template paths
	templatePath := fmt.Sprintf("/api/teams/%s/templates/%s", url.PathEscape(teamSlug), url.PathEscape(templateName))
	publicTemplatePath := fmt.Sprintf("/api/public/templates/%s/%s", url.PathEscape(teamSlug), url.PathEscape(templateName))

	if version != "" {
		templatePath = fmt.Sprintf("%s/v/%s", templatePath, url.PathEscape(version))
		publicTemplatePath = fmt.Sprintf("%s/v/%s", publicTemplatePath, url.PathEscape(version))
	}

	// Try authenticated team endpoint first
	template, err := c.tryTemplateEndpoint(templatePath)
	if err == nil {
		return template, nil
	}

	// If authenticated endpoint fails, try public endpoint
	template, publicErr := c.tryTemplateEndpoint(publicTemplatePath)
	if publicErr != nil {
		// If both fail, return the original error
		return nil, err
	}

	return template, nil
}
