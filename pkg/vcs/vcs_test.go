package vcs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVCS(t *testing.T) {
	// Because this test gets ran from potentially different working dirs we do everything relative to the curDir and repoRoot
	curDir, err := os.Getwd()
	require.NoError(t, err)
	repo, err := git.PlainOpenWithOptions(curDir, &git.PlainOpenOptions{DetectDotGit: true})
	require.NoError(t, err)
	w, err := repo.Worktree()
	require.NoError(t, err)
	repoRoot := w.Filesystem.Root()
	fmt.Println(repoRoot)
	dir, err := filepath.Rel(repoRoot, curDir)
	require.NoError(t, err)

	type args struct {
		acornfile    string
		buildContext string
	}
	tests := []struct {
		name     string
		args     args
		expected args
	}{
		{
			name: "simple",
			args: args{
				acornfile:    "Acornfile",
				buildContext: ".",
			},
			expected: args{
				acornfile:    filepath.Join(dir, "Acornfile"),
				buildContext: dir,
			},
		},
		{
			name: "given filename",
			args: args{
				acornfile:    "test.acorn",
				buildContext: ".",
			},
			expected: args{
				acornfile:    filepath.Join(dir, "test.acorn"),
				buildContext: dir,
			},
		},
		{
			name: "given nested filename",
			args: args{
				acornfile:    "nested/test.acorn",
				buildContext: ".",
			},
			expected: args{
				acornfile:    filepath.Join(dir, "nested/test.acorn"),
				buildContext: dir,
			},
		},
		{
			name: "given build context",
			args: args{
				acornfile:    "Acornfile",
				buildContext: "nested",
			},
			expected: args{
				acornfile:    filepath.Join(dir, "Acornfile"),
				buildContext: filepath.Join(dir, "nested"),
			},
		},
		{
			name: "given build context and filename",
			args: args{
				acornfile:    "test.acorn",
				buildContext: "nested",
			},
			expected: args{
				acornfile:    filepath.Join(dir, "test.acorn"),
				buildContext: filepath.Join(dir, "nested"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcs := VCS(tt.args.acornfile, tt.args.buildContext)
			assert.NotEqual(t, "", vcs.Revision)
			assert.Equal(t, tt.expected.acornfile, vcs.Acornfile)
			assert.Equal(t, tt.expected.buildContext, vcs.BuildContext)
		})
	}
}
