package models

// Describer provides Name and Description for embedding into
// concrete check and action types.
type Describer struct {
	name        string
	description string
}

func NewDescriber(name, description string) Describer {
	return Describer{
		name:        name,
		description: description,
	}
}

// Name returns the registered name.
func (d Describer) Name() string { return d.name }

// Description returns the human-readable description.
func (d Describer) Description() string { return d.description }
