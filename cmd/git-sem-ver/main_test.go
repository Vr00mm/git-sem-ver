package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Vr00mm/git-sem-ver/internal/gitctx"
	"github.com/Vr00mm/git-sem-ver/internal/semver"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// miniRepo creates a minimal git repo in dir with one commit and optional tag.
func miniRepo(t *testing.T, dir string, tagName string) {
	t.Helper()
	r, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)

	w, err := r.Worktree()
	require.NoError(t, err)

	p := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(p, []byte("content"), 0o600))
	_, err = w.Add("file.txt")
	require.NoError(t, err)

	hash, err := w.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "T", Email: "t@t.com", When: time.Now()},
	})
	require.NoError(t, err)

	if tagName != "" {
		_, err = r.CreateTag(tagName, hash, nil)
		require.NoError(t, err)
	}
}

// ── bump ─────────────────────────────────────────────────────────────────────

func TestBump(t *testing.T) {
	t.Parallel()

	base := semver.Version{Major: 1, Minor: 2, Patch: 3}

	tests := []struct {
		name string
		kind gitctx.BumpKind
		want semver.Version
	}{
		{"patch", gitctx.BumpPatch, semver.Version{Major: 1, Minor: 2, Patch: 4}},
		{"minor", gitctx.BumpMinor, semver.Version{Major: 1, Minor: 3, Patch: 0}},
		{"major", gitctx.BumpMajor, semver.Version{Major: 2, Minor: 0, Patch: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, bump(base, tt.kind))
		})
	}
}

func TestBumpFromZero(t *testing.T) {
	t.Parallel()

	zero := semver.Version{}
	assert.Equal(t, semver.Version{Major: 0, Minor: 0, Patch: 1}, bump(zero, gitctx.BumpPatch))
	assert.Equal(t, semver.Version{Major: 0, Minor: 1, Patch: 0}, bump(zero, gitctx.BumpMinor))
	assert.Equal(t, semver.Version{Major: 1, Minor: 0, Patch: 0}, bump(zero, gitctx.BumpMajor))
}

// ── preRelease ───────────────────────────────────────────────────────────────

func TestPreRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  gitctx.Context
		n    int
		want string
	}{
		{"main", gitctx.Context{Kind: gitctx.KindMain}, 7, "rc.7"},
		{"develop", gitctx.Context{Kind: gitctx.KindDevelop}, 3, "dev.3"},
		{"feature", gitctx.Context{Kind: gitctx.KindFeature, ShortName: "new-login"}, 5, "feat.new-login.5"},
		{"fix", gitctx.Context{Kind: gitctx.KindFix, ShortName: "crash"}, 2, "fix.crash.2"},
		{"hotfix", gitctx.Context{Kind: gitctx.KindHotfix, ShortName: "null-ptr"}, 1, "hotfix.null-ptr.1"},
		{"release", gitctx.Context{Kind: gitctx.KindRelease}, 4, "beta.4"},
		{"other with name", gitctx.Context{Kind: gitctx.KindOther, ShortName: "chore-deps"}, 9, "branch.chore-deps.9"},
		{"other without name", gitctx.Context{Kind: gitctx.KindOther, ShortName: ""}, 2, "branch.2"},
		{"zero commits", gitctx.Context{Kind: gitctx.KindMain}, 0, "rc.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, preRelease(tt.ctx, tt.n))
		})
	}
}

// ── writeGithubOutput ────────────────────────────────────────────────────────

func TestWriteGithubOutput(t *testing.T) {
	t.Run("writes version to GITHUB_OUTPUT file", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "github-output-*")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		t.Setenv("GITHUB_OUTPUT", f.Name())

		require.NoError(t, writeGithubOutput("1.3.0-feat.new-api.5"))

		content, err := os.ReadFile(f.Name())
		require.NoError(t, err)
		assert.Equal(t, "version=1.3.0-feat.new-api.5\n", string(content))
	})

	t.Run("no-op when GITHUB_OUTPUT is unset", func(t *testing.T) {
		t.Setenv("GITHUB_OUTPUT", "")
		assert.NoError(t, writeGithubOutput("1.0.0"))
	})

	t.Run("appends to existing file content", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "github-output-*")
		require.NoError(t, err)
		_, err = f.WriteString("other-output=foo\n")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		t.Setenv("GITHUB_OUTPUT", f.Name())
		require.NoError(t, writeGithubOutput("2.0.0"))

		content, err := os.ReadFile(f.Name())
		require.NoError(t, err)
		assert.Equal(t, "other-output=foo\nversion=2.0.0\n", string(content))
	})

	t.Run("returns error when output file cannot be opened", func(t *testing.T) {
		t.Setenv("GITHUB_OUTPUT", "/nonexistent/dir/output")
		err := writeGithubOutput("1.0.0")
		require.Error(t, err)
	})
}

// ── run ──────────────────────────────────────────────────────────────────────

func TestRun(t *testing.T) {
	t.Run("tag: returns clean version", func(t *testing.T) {
		t.Setenv("GITHUB_REF_TYPE", "tag")
		t.Setenv("GITHUB_REF_NAME", "v2.5.0")
		t.Setenv("GIT_SEM_VER_BUMP", "")

		got, err := run()
		require.NoError(t, err)
		assert.Equal(t, "2.5.0", got)
	})

	t.Run("tag: invalid semver name returns error", func(t *testing.T) {
		t.Setenv("GITHUB_REF_TYPE", "tag")
		t.Setenv("GITHUB_REF_NAME", "not-a-version")
		t.Setenv("GIT_SEM_VER_BUMP", "")

		_, err := run()
		require.Error(t, err)
	})

	t.Run("branch: no git repo returns error", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv("GITHUB_REF_TYPE", "branch")
		t.Setenv("GITHUB_REF_NAME", "main")
		t.Setenv("GIT_SEM_VER_BUMP", "")

		_, err := run()
		require.Error(t, err)
	})

	t.Run("branch: no prior tag uses zero base", func(t *testing.T) {
		dir := t.TempDir()
		miniRepo(t, dir, "")
		t.Chdir(dir)
		t.Setenv("GITHUB_REF_TYPE", "branch")
		t.Setenv("GITHUB_REF_NAME", "feat/new-thing")
		t.Setenv("GIT_SEM_VER_BUMP", "")

		got, err := run()
		require.NoError(t, err)
		// no tags, 1 commit, feat → minor bump from 0.0.0 → 0.1.0
		assert.Equal(t, "0.1.0-feat.new-thing.1", got)
	})

	t.Run("branch: version bumped from latest tag", func(t *testing.T) {
		dir := t.TempDir()
		miniRepo(t, dir, "v1.2.3")
		t.Chdir(dir)
		t.Setenv("GITHUB_REF_TYPE", "branch")
		t.Setenv("GITHUB_REF_NAME", "main")
		t.Setenv("GIT_SEM_VER_BUMP", "")

		got, err := run()
		require.NoError(t, err)
		// tag at HEAD → 0 commits → patch bump → 1.2.4-rc.0
		assert.Equal(t, "1.2.4-rc.0", got)
	})
}

// ── execute ──────────────────────────────────────────────────────────────────

func TestExecute(t *testing.T) {
	t.Run("success: version written to stdout", func(t *testing.T) {
		t.Setenv("GITHUB_REF_TYPE", "tag")
		t.Setenv("GITHUB_REF_NAME", "v1.5.0")
		t.Setenv("GIT_SEM_VER_BUMP", "")
		t.Setenv("GITHUB_OUTPUT", "")

		var out, errOut strings.Builder
		err := execute(&out, &errOut)

		require.NoError(t, err)
		assert.Equal(t, "1.5.0\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("error: message written to stderr", func(t *testing.T) {
		t.Setenv("GITHUB_REF_TYPE", "tag")
		t.Setenv("GITHUB_REF_NAME", "invalid")
		t.Setenv("GIT_SEM_VER_BUMP", "")

		var out, errOut strings.Builder
		err := execute(&out, &errOut)

		require.Error(t, err)
		assert.Empty(t, out.String())
		assert.Contains(t, errOut.String(), "error:")
	})

	t.Run("writeGithubOutput failure: warning on stderr, no error returned", func(t *testing.T) {
		t.Setenv("GITHUB_REF_TYPE", "tag")
		t.Setenv("GITHUB_REF_NAME", "v1.0.0")
		t.Setenv("GIT_SEM_VER_BUMP", "")
		t.Setenv("GITHUB_OUTPUT", "/nonexistent/dir/output")

		var out, errOut strings.Builder
		err := execute(&out, &errOut)

		require.NoError(t, err)
		assert.Equal(t, "1.0.0\n", out.String())
		assert.Contains(t, errOut.String(), "warning:")
	})
}

// ── main ─────────────────────────────────────────────────────────────────────

func TestMain_success(t *testing.T) {
	t.Setenv("GITHUB_REF_TYPE", "tag")
	t.Setenv("GITHUB_REF_NAME", "v3.0.0")
	t.Setenv("GIT_SEM_VER_BUMP", "")
	t.Setenv("GITHUB_OUTPUT", "")

	// execute() returns nil → osExit is not called; covers the success branch.
	main()
}

func TestMain_callsExitOnError(t *testing.T) {
	t.Setenv("GITHUB_REF_TYPE", "tag")
	t.Setenv("GITHUB_REF_NAME", "not-semver")
	t.Setenv("GIT_SEM_VER_BUMP", "")

	var captured int
	osExit = func(code int) { captured = code }
	defer func() { osExit = os.Exit }()

	main()

	assert.Equal(t, 1, captured)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// Ensure miniRepo compiles — use plumbing to prevent unused import warnings.
var _ = fmt.Sprintf
var _ plumbing.Hash
