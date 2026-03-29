// SPDX-License-Identifier: Apache-2.0

package githubactions

import (
	"context"
	"fmt"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/github"
)

// ActionSetRepoDescription updates the repository description on GitHub.
// The description is taken from the "description" param of the first subject.
type ActionSetRepoDescription struct {
	engine.GitHubAction
}

func (ActionSetRepoDescription) ApplyTo() engine.SubjectKind { return engine.SubjectRepo }

func (a ActionSetRepoDescription) Execute(
	ctx context.Context,
	client *github.Client,
	data *github.RepoData,
	_ []string,
	params []map[string]string,
) (engine.Result, error) {
	if data == nil {
		return engine.Result{}, fmt.Errorf("github repo data is required")
	}

	// Get description from params.
	var description string
	if len(params) > 0 && params[0] != nil {
		description = params[0]["description"]
	}

	if description == "" {
		return engine.Result{}, fmt.Errorf("no description provided (set 'description' param)")
	}

	if err := client.SetDescription(ctx, data.Owner, data.Repo, description); err != nil {
		return engine.Result{
			OK:      false,
			Message: fmt.Sprintf("failed to set description on %s: %v", data.FullName, err),
		}, err
	}

	return engine.Result{
		OK:      true,
		Message: fmt.Sprintf("set description on %s", data.FullName),
	}, nil
}
