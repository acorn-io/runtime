package vcs

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
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

func AcornfileFromApp(ctx context.Context, app *apiv1.App) (string, error) {

	vcs := app.Status.Staged.AppImage.VCS

	if len(vcs.Remotes) == 0 {
		return "", fmt.Errorf("clone can only be done on an app built from a git repository")
	}

	auth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		return "", err
	}

	for _, remote := range vcs.Remotes {
		var gitUrl string
		httpUrl, err := url.Parse(remote)
		if err == nil {
			gitUrl = fmt.Sprintf("git@%s:%s", httpUrl.Host, httpUrl.Path[1:])
		} else {
			gitUrl = remote
		}

		// TODO workdir named after git repo, cloned app name, or just this app's name?
		idx := strings.LastIndex(gitUrl, "/")
		if idx < 0 || idx >= len(gitUrl) {
			fmt.Printf("failed to determine repository name %q\n", gitUrl)
			continue
		}
		workdir := filepath.Clean(strings.TrimSuffix(gitUrl[idx+1:], ".git"))

		// Clone git repo
		_, err = git.PlainCloneContext(ctx, workdir, false, &git.CloneOptions{
			URL:      gitUrl,
			Progress: os.Stderr,
			Auth:     auth,
		})
		// TODO handle ErrRepositoryAlreadyExists some way
		if err != nil {
			fmt.Printf("failed to clone repository %q: %s\n", gitUrl, err.Error())
			continue
		}

		acornfile := filepath.Join(workdir, vcs.Acornfile)
		// TODO if acornfile exists but is different than what is cached should we overwrite?
		if _, err := os.Stat(acornfile); errors.Is(err, os.ErrNotExist) {
			// Acornfile does not exist so we should create it
			err = os.WriteFile(acornfile, []byte(app.Status.Staged.AppImage.Acornfile), 0666)
			if err != nil {
				fmt.Printf("failed to create file %q in repository %q: %s", acornfile, gitUrl, err.Error())
				// TODO we hit an error state but already cloned the repo, should we clean up the repo we cloned?
				continue
			}
		} else {
			fmt.Printf("could not check for file %q in repository %q: %s", acornfile, gitUrl, err.Error())
			// TODO we hit an error state but already cloned the repo, should we clean up the repo we cloned?
			continue
		}
		return acornfile, nil
	}
	return "", fmt.Errorf("failed to resolve an acornfile from the app")
}
