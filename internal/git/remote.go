package git

import (
	"bufio"
	"context"
	"strings"
)

// Remote represents a git remote with its name and URL.
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

// Remotes runs git remote -v and returns the parsed remotes.
func (r *Runner) Remotes(ctx context.Context) ([]Remote, error) {
	out, err := r.run(ctx, cmdRemoteVerbose()...)
	if err != nil {
		return nil, err
	}

	return parseRemotes(out), nil
}

// RemoteMap runs git remote -v and returns a map of remote name to fetch URL.
//
// This is a convenience method matching the Remotes map[string]string field
// in the config.Repository type.
func (r *Runner) RemoteMap(ctx context.Context) (map[string]string, error) {
	remotes, err := r.Remotes(ctx)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string, len(remotes))
	for _, rm := range remotes {
		m[rm.Name] = rm.FetchURL
	}

	return m, nil
}

// parseRemotes parses the output of git remote -v.
//
// Each line has the form:
//
//	origin	https://github.com/user/repo.git (fetch)
//	origin	https://github.com/user/repo.git (push)
func parseRemotes(output string) []Remote {
	// Collect into a map so we can merge fetch+push lines.
	type urls struct {
		fetch string
		push  string
	}

	seen := make(map[string]*urls)
	var order []string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		name, rest, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}

		url, kind, ok := strings.Cut(rest, " ")
		if !ok {
			continue
		}

		u, exists := seen[name]
		if !exists {
			u = &urls{}
			seen[name] = u
			order = append(order, name)
		}

		switch kind {
		case "(fetch)":
			u.fetch = url
		case "(push)":
			u.push = url
		}
	}

	remotes := make([]Remote, 0, len(order))
	for _, name := range order {
		u := seen[name]
		remotes = append(remotes, Remote{
			Name:     name,
			FetchURL: u.fetch,
			PushURL:  u.push,
		})
	}

	return remotes
}
