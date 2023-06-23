package vcs

import (
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"gopkg.in/src-d/go-git.v4"
)

func VCS(path string) (result v1.VCS) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return
	}
	w, err := repo.Worktree()
	if err != nil {
		return
	}
	s, err := w.Status()
	if err != nil {
		return
	}
	head, err := repo.Head()
	if err != nil {
		return
	}

	result = v1.VCS{
		Revision: head.Hash().String(),
		Modified: !s.IsClean(),
	}

	// Set optional remotes field
	remotes, err := repo.Remotes()
	if err != nil {
		return
	}

	for _, remote := range remotes {
		result.Remotes = append(result.Remotes, remote.Config().URLs...)
	}

	return
}
