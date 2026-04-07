package engine

type engineError string

func (e engineError) Error() string {
	return string(e)
}

const (
	ErrEngine         engineError = "error git-janitor engine"
	ErrNotImplemented engineError = "not implemented"
)
