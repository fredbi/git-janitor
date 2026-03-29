// SPDX-License-Identifier: Apache-2.0

package githubchecks

import (
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/github"
)

// CheckSecurityNotEnabled detects when none of the GitHub security scanners
// are accessible for a repository. This may indicate the scanners are not
// enabled (on repos you own) or that the token lacks access (on third-party repos).
type CheckSecurityNotEnabled struct {
	engine.GitHubCheck
}

func (c CheckSecurityNotEnabled) Evaluate(data *github.RepoData) (iter.Seq[engine.Alert], error) {
	// Security not queried by config — skip silently.
	if data.SecuritySkipped {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	// If at least one scanner is accessible (>= 0), security is partially enabled.
	if data.DependabotAlerts >= 0 || data.CodeScanningAlerts >= 0 || data.SecretScanningAlerts >= 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	// Distinguish: repos you can admin → "not enabled" (actionable),
	// repos you don't own → "no access" (informational).
	if data.HasAdminAccess {
		settingsURL := data.HTMLURL + "/settings/security_analysis"

		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityMedium,
			Summary:   "no security scanners enabled",
			Detail:    fmt.Sprintf("No security scanners are enabled on %s. Enable them at %s", data.FullName, settingsURL),
			Suggestions: []engine.ActionSuggestion{{
				ActionName:  "open-in-browser",
				SubjectKind: engine.SubjectRepo,
				Subjects:    []string{settingsURL},
			}},
		}), nil
	}

	// No admin access — likely a third-party repo. Just informational.
	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityInfo,
		Summary:   "no access to security APIs",
		Detail:    fmt.Sprintf("Security APIs are not accessible for %s — token likely lacks permissions on this repository.", data.FullName),
	}), nil
}

// CheckSecurityAlerts detects open security alerts reported by GitHub's
// security tools (Dependabot, code scanning, secret scanning).
type CheckSecurityAlerts struct {
	engine.GitHubCheck
}

func (c CheckSecurityAlerts) Evaluate(data *github.RepoData) (iter.Seq[engine.Alert], error) {
	if data.SecuritySkipped {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	total := data.SecurityAlerts()

	// Not fetched (all -1): skip silently.
	if total < 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	if total == 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
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

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityHigh,
		Summary:   summary,
		Detail:    detail,
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "open-in-browser",
			SubjectKind: engine.SubjectRepo,
			Subjects:    []string{securityURL},
		}},
	}), nil
}
