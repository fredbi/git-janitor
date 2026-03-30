package engine

type errEngine string

func (e errEngine) Error() string {
	return string(e)
}

const (
	ErrEngine         errEngine = "error git-janitor engine"
	ErrNotImplemented errEngine = "not implemented"
)
