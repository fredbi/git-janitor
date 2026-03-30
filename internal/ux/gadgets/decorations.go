package gadgets

import (
	"github.com/fredbi/git-janitor/internal/models"
)

// SeverityBullet returns a colored emoji for the given engine.Severity.
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
