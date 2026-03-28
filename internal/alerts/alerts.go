package alerts

import "context"

type Alert interface {
	Name() string
	Description() string
	Check(ctx context.Context, repo string) (*AlertResult, error)
}

type AlertResult struct {
	Err      error
	Repo     string
	Message  string
	Args     string
	Severity Severity
}

func (r AlertResult) SuggestedActions() []string {
	return nil // TODO
}

// Severity represents the urgency level of an alert.
type Severity int

const (
	SeverityNone   Severity = iota // no bullet
	SeverityLow                    // yellow bullet
	SeverityMedium                 // orange bullet
	SeverityHigh
)
