// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// SecurityNotEnabled detects when none of the GitHub security scanners
// are accessible for a repository. This may indicate the scanners are not
// enabled (on repos you own) or that the token lacks access (on third-party repos).
type SecurityNotEnabled struct {
	githubCheck
}

func NewSecurityNotEnabled() SecurityNotEnabled {
	return SecurityNotEnabled{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-security-not-enabled",
				"detects when no security scanners are accessible",
			),
		},
	}
}

func (c SecurityNotEnabled) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c SecurityNotEnabled) evaluate(data *backend.RepoInfo) (iter.Seq[models.Alert], error) {
	// Security not queried by config — skip silently.
	if data.SecuritySkipped {
		return noAlert(c.Name())
	}

	// If at least one scanner is accessible (>= 0), security is partially enabled.
	if data.DependabotAlerts >= 0 || data.CodeScanningAlerts >= 0 || data.SecretScanningAlerts >= 0 {
		return noAlert(c.Name())
	}

	// Distinguish: repos you can admin → "not enabled" (actionable),
	// repos you don't own → "no access" (informational).
	if data.HasAdminAccess {
		settingsURL := data.HTMLURL + "/settings/security_analysis"
		suggestion := repoSuggestion("open-in-browser", simpleSubject(settingsURL))

		return singleAlert(models.Alert{
			CheckName:   c.Name(),
			Severity:    models.SeverityMedium,
			Summary:     "no security scanners enabled",
			Detail:      fmt.Sprintf("No security scanners are enabled on %s. Enable them at %s", data.FullName, settingsURL),
			Suggestions: []models.ActionSuggestion{suggestion},
		}), nil
	}

	// No admin access — likely a third-party repo. Just informational.
	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "no access to security APIs",
		Detail:    fmt.Sprintf("Security APIs are not accessible for %s — token likely lacks permissions on this repository.", data.FullName),
	}), nil
}

// SecurityAlerts detects open security alerts reported by GitHub's
// security tools (Dependabot, code scanning, secret scanning).
type SecurityAlerts struct {
	githubCheck
}

func NewSecurityAlerts() SecurityAlerts {
	return SecurityAlerts{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-security-alerts",
				"detects open security alerts (Dependabot, code scanning, secret scanning)",
			),
		},
	}
}

func (c SecurityAlerts) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c SecurityAlerts) evaluate(data *backend.RepoInfo) (iter.Seq[models.Alert], error) {
	if data.SecuritySkipped {
		return noAlert(c.Name())
	}

	total := data.SecurityAlerts()

	// Not fetched (all -1): skip silently.
	if total <= 0 {
		return noAlert(c.Name())
	}

	// Build a breakdown of which scanners found alerts.
	var parts []string
	if data.DependabotAlerts > 0 {
		parts = append(parts, fmt.Sprintf("%d dependabot", data.DependabotAlerts))
	}

	if data.CodeScanningAlerts > 0 {
		parts = append(parts, fmt.Sprintf("%d code scanning", data.CodeScanningAlerts))
	}

	if data.SecretScanningAlerts > 0 {
		parts = append(parts, fmt.Sprintf("%d secret scanning", data.SecretScanningAlerts))
	}

	securityURL := data.HTMLURL + "/security"
	summary := fmt.Sprintf("%d open security alert(s)", total)
	detail := fmt.Sprintf("%s: %s. Review at %s",
		data.FullName, strings.Join(parts, ", "), securityURL)
	suggestion := repoSuggestion("open-in-browser", simpleSubject(securityURL))

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityHigh,
		Summary:     summary,
		Detail:      detail,
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}
