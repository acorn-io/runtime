package vcs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/go-git/go-git/v5"
)

func VCS(filePath, buildContextPath string) (result v1.VCS) {
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return
	}
	buildContextPath, err = filepath.Abs(buildContextPath)
	if err != nil {
		return
	}
	repo, err := git.PlainOpenWithOptions(filePath, &git.PlainOpenOptions{
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

	acornfile, err := filepath.Rel(w.Filesystem.Root(), filePath)
	if err != nil {
		return
	}
	buildContext, err := filepath.Rel(w.Filesystem.Root(), buildContextPath)
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

func ImageInfoFromApp(ctx context.Context, app *apiv1.App, cloneDir string) (string, string, error) {
	vcs := app.Status.AppImage.VCS
	if len(vcs.Remotes) == 0 {
		return "", "", fmt.Errorf("clone can only be done on an app built from a git repository")
	}
	if vcs.Acornfile == "" || vcs.BuildContext == "" {
		return "", "", fmt.Errorf("app is missing required vcs information, image must be rebuilt with a newer acorn cli")
	}

	workdir := filepath.Join(cloneDir, app.Name)

	for i, remote := range vcs.Remotes {
		// Check for the directory we want to use
		f, err := os.Open(workdir)
		if err == nil {
			// Directory exists, check if empty
			_, err = f.ReadDir(1)
			if err != nil {
				// Directory is empty, clone the repo
				err = gitClone(ctx, workdir, remote)
				if err != nil {
					fmt.Printf("%v\n", err)
					continue
				}
			} else {
				// Directory is not empty so we assume it exists, add a new remote and try to fetch
				remoteName := fmt.Sprintf("remote%d", i)
				err = gitRemoteAdd(ctx, workdir, remoteName, remote)
				if err != nil {
					fmt.Printf("%v\n", err)
				}
				err = gitFetch(ctx, workdir, remoteName)
				if err != nil {
					fmt.Printf("%v\n", err)
					continue
				}
			}
		} else if os.IsNotExist(err) {
			// Directory does not exist, just clone to create it
			err = gitClone(ctx, workdir, remote)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
		} else {
			fmt.Printf("failed to check for the existence of directory %q: %v", workdir, err)
			continue
		}

		// Try to checkout the revision
		err = gitCheckout(ctx, workdir, vcs.Revision)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		// Create the Acornfile in the repository
		acornfile := filepath.Join(workdir, vcs.Acornfile)
		err = os.WriteFile(acornfile, []byte(app.Status.AppImage.Acornfile), 0666)
		if err != nil {
			fmt.Printf("failed to create file %q in repository %q: %v\n", acornfile, workdir, err)
			continue
		}

		// Determine if the Acornfile is dirty or not
		if gitDirty(ctx, workdir) {
			fmt.Printf("NOTE: The Acornfile used for this acorn differs from the git repository. Run `git status` for more details.")
		}

		// Get the build context
		buildContext := filepath.Join(workdir, vcs.BuildContext)

		return acornfile, buildContext, nil
	}
	return "", "", fmt.Errorf("failed to resolve an acornfile from the app")
}

func gitClone(ctx context.Context, workdir, remote string) (err error) {
	args := []string{"clone", remote, workdir}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmdErr := cmd.Run()
	if cmdErr != nil {
		err = fmt.Errorf("failed to clone repository %q: %v", remote, err)
	}
	return
}

func gitRemoteAdd(ctx context.Context, workdir, remoteName, remote string) (err error) {
	args := []string{"-C", workdir, "remote", "add", remoteName, remote}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmdErr := cmd.Run()
	if cmdErr != nil {
		err = fmt.Errorf("failed to add remote %q to repository %q: %v", remote, workdir, err)
	}
	return
}

func gitFetch(ctx context.Context, workdir, remote string) (err error) {
	args := []string{"-C", workdir, "fetch", remote}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmdErr := cmd.Run()
	if cmdErr != nil {
		err = fmt.Errorf("failed to fetch remote %q in repository %q: %v", remote, workdir, err)
	}
	return
}

func gitCheckout(ctx context.Context, workdir, revision string) (err error) {
	args := []string{"-C", workdir, "checkout", revision}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmdErr := cmd.Run()
	if cmdErr != nil {
		err = fmt.Errorf("failed to checkout revision %q: %v", revision, err)
	}
	return
}

func gitDirty(ctx context.Context, workdir string) bool {
	args := []string{"-C", workdir, "diff", "--quiet"}
	cmd := exec.CommandContext(ctx, "git", args...)
	return cmd.Run() != nil
}
