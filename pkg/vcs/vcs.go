package vcs

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func VCS(filePath, buildContextPath string) (result v1.VCS) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return
	}
	buildContextAbs, err := filepath.Abs(buildContextPath)
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

	acornfile, err := filepath.Rel(w.Filesystem.Root(), absPath)
	if err != nil {
		return
	}
	buildContext, err := filepath.Rel(w.Filesystem.Root(), buildContextAbs)
	if err != nil {
		return
	}

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
		Revision:     head.Hash().String(),
		Clean:        !modified && !untracked,
		Modified:     modified,
		Untracked:    untracked,
		Acornfile:    acornfile,
		BuildContext: buildContext,
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

func ImageInfoFromApp(ctx context.Context, app *apiv1.App) (string, string, error) {
	vcs := app.Status.Staged.AppImage.VCS
	if len(vcs.Remotes) == 0 {
		return "", "", fmt.Errorf("clone can only be done on an app built from a git repository")
	}
	if vcs.Acornfile == "" {
		return "", "", fmt.Errorf("app has no acornfile information in vcs")
	}

	// Create auth object to use when fetching and cloning git repos
	auth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		return "", "", err
	}

	for _, remote := range vcs.Remotes {
		// Since we use ssh auth to clone the repo we need a git url but will sometimes get http urls
		var gitUrl string
		httpUrl, err := url.Parse(remote)
		if err == nil {
			gitUrl = fmt.Sprintf("git@%s:%s", httpUrl.Host, httpUrl.Path[1:])
		} else {
			gitUrl = remote
		}

		// Determine the repository name from the repo url
		idx := strings.LastIndex(remote, "/")
		if idx < 0 || idx >= len(remote) {
			fmt.Printf("failed to determine repository name %q\n", remote)
			continue
		}
		workdir := filepath.Clean(strings.TrimSuffix(remote[idx+1:], ".git"))

		// Clone git repo and checkout revision
		fmt.Printf("# Cloning repository %q into directory %q\n", gitUrl, workdir)
		repo, err := git.PlainCloneContext(ctx, workdir, false, &git.CloneOptions{
			URL:      gitUrl,
			Auth:     auth,
			Progress: os.Stderr,
		})
		if err != nil {
			fmt.Printf("failed to clone repository %q: %s\n", gitUrl, err.Error())
			continue
		}
		w, err := repo.Worktree()
		if err != nil {
			fmt.Printf("failed to get worktree from repository %q: %s\n", workdir, err.Error())
			continue
		}
		err = w.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(vcs.Revision),
		})
		if err != nil {
			fmt.Printf("failed to checkout revision %q for repository %q: %s\n", vcs.Revision, workdir, err.Error())
			continue
		}

		// Create the Acornfile in the repository
		acornfile := filepath.Join(workdir, vcs.Acornfile)
		err = os.WriteFile(acornfile, []byte(app.Status.Staged.AppImage.Acornfile), 0666)
		if err != nil {
			fmt.Printf("failed to create file %q in repository %q: %s\n", acornfile, workdir, err.Error())
			continue
		}

		// Determine if the Acornfile is dirty or not
		s, err := w.Status()
		if err == nil {
			if !s.IsClean() {
				fmt.Printf("running with a dirty Acornfile %q\n", acornfile)
			}
		} else {
			fmt.Printf("failed to get status from worktree %q: %s\n", workdir, err.Error())
		}

		// Get the build context
		buildContext := filepath.Join(workdir, vcs.BuildContext)

		return acornfile, buildContext, nil
	}
	return "", "", fmt.Errorf("failed to resolve an acornfile from the app")
}
