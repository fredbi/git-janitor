package gadgets

import (
	"fmt"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// SeverityBullet returns a colored emoji for the given models.Severity.
func SeverityBullet(s models.Severity) string {
	switch s {
	case models.SeverityCritical:
		return "💀"
	case models.SeverityHigh:
		return "🔴"
	case models.SeverityMedium:
		return "🟠"
	case models.SeverityLow:
		return "🟡"
	case models.SeverityInfo:
		return "🔵"
	default:
		return "⚪"
	}
}

func DestructiveWarning() string {
	return "⚠️  Run DESTRUCTIVE"
}

// TimeAgo returns a human-readable relative time string.
func TimeAgo(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 min ago"
		}

		return fmt.Sprintf("%d min ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24) //nolint:mnd // 24 hours per day

		switch {
		case days == 1:
			return "1 day ago"
		case days < 30: //nolint:mnd // month threshold
			return fmt.Sprintf("%dd ago", days)
		case days < 365: //nolint:mnd // year threshold
			months := days / 30 //nolint:mnd // approximate
			if months == 1 {
				return "1 month ago"
			}

			return fmt.Sprintf("%dmo ago", months)
		default:
			years := days / 365 //nolint:mnd // approximate
			if years == 1 {
				return "1 year ago"
			}

			return fmt.Sprintf("%dy ago", years)
		}
	}
}

// ElideLongLabel truncates a string to maxWidth, appending "..." if needed.
func ElideLongLabel(str string) string {
	const (
		maxWidth = 40
		elided   = "..."
	)

	if len(str) > maxWidth {
		return str[:maxWidth-len(elided)] + elided
	}

	return str
}
