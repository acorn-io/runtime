package vcs

import (
	"path/filepath"
	"strings"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/go-git/go-git/v5"
)

func VCS(filePath string) (result v1.VCS) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return
	}
	repo, err := git.PlainOpenWithOptions(filepath.Dir(absPath), &git.PlainOpenOptions{
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

	var sb strings.Builder
	sb.WriteString(w.Filesystem.Root())
	sb.WriteRune(filepath.Separator)
	acornfile := strings.TrimPrefix(absPath, sb.String())

	var (
		modified, untracked bool
	)
	for _, status := range s {
		if status.Worktree == git.Untracked {
			untracked = true
			continue
		}
		if status.Worktree != git.Unmodified || status.Staging != git.Unmodified {
			modified = true
			continue
		}
	}

	result = v1.VCS{
		Revision:  head.Hash().String(),
		Clean:     !modified && !untracked,
		Modified:  modified,
		Untracked: untracked,
		Acornfile: acornfile,
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
