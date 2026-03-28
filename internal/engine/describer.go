package engine

// Describer provides Name and Description for embedding into
// concrete check and action types.
type Describer struct {
	CheckName        string
	CheckDescription string
}

// Name returns the registered name.
func (d Describer) Name() string { return d.CheckName }

// Description returns the human-readable description.
func (d Describer) Description() string { return d.CheckDescription }
