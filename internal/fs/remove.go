package fs

import (
	"context"
	"errors"
	"os"

	"github.com/fredbi/git-janitor/internal/models"
)

func DeleteLocalRepo(_ context.Context, info *models.RepoInfo) error {
	if info == nil {
		return errors.New("repo info is required to delete local clone")
	}

	path := info.Path
	if path == "" {
		return errors.New("empty repo path, refusing to delete")
	}

	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return nil
}
