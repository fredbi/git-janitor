package registry

type registryError string

func (e registryError) Error() string {
	return string(e)
}

const ErrDuplicate registryError = "registered items must be named uniquely across the board"
