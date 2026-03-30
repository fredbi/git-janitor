package engine

import (
	gitrepo "github.com/fredbi/git-janitor/internal/git/backend"
	githubrepo "github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.RepoInfo = &RepoInfo{}

type RepoInfo struct {
	GitInfo    *gitrepo.RepoInfo
	GitHubInfo *githubrepo.RepoInfo
}

func (*RepoInfo) IsRepoInfo() {}

func (r *RepoInfo) IsEmpty() bool {
	return r == nil || r.GitInfo == nil
}

func (r *RepoInfo) Err() error {
	if r == nil || r.GitInfo == nil {
		return nil
	}

	return r.GitInfo.Err
}

func NoGit(pth string) *RepoInfo {
	return &RepoInfo{
		GitInfo: &gitrepo.RepoInfo{
			Path:  pth,
			IsGit: false,
			SCM:   models.SCMNone,
			Kind:  models.RepoKindNotGit,
		},
	}
}

func OriginFetchURL(info *RepoInfo) string {
	if info.IsEmpty() {
		return ""
	}

	return gitrepo.OriginFetchURL(info.GitInfo.Remotes)
}
