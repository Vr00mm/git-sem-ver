package git_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Vr00mm/git-sem-ver/internal/git"
	"github.com/Vr00mm/git-sem-ver/internal/semver"
)

// repo is a test helper that builds a git repository in a temp directory.
type repo struct {
	t    *testing.T
	r    *gogit.Repository
	dir  string
	seq  int
	base time.Time
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	dir := t.TempDir()
	r, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)
	return &repo{t: t, r: r, dir: dir, base: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
}

// commit creates a unique file, stages it, and commits it. Returns the commit hash.
func (r *repo) commit(msg string) plumbing.Hash {
	r.t.Helper()
	r.seq++

	path := filepath.Join(r.dir, fmt.Sprintf("file-%03d.txt", r.seq))
	require.NoError(r.t, os.WriteFile(path, []byte(msg), 0o600))

	w, err := r.r.Worktree()
	require.NoError(r.t, err)

	_, err = w.Add(fmt.Sprintf("file-%03d.txt", r.seq))
	require.NoError(r.t, err)

	hash, err := w.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			// Increment time per commit so walk order is deterministic.
			When: r.base.Add(time.Duration(r.seq) * time.Second),
		},
	})
	require.NoError(r.t, err)
	return hash
}

// tag creates a lightweight tag pointing to hash.
func (r *repo) tag(hash plumbing.Hash, name string) {
	r.t.Helper()
	_, err := r.r.CreateTag(name, hash, nil)
	require.NoError(r.t, err)
}

// annotatedTag creates an annotated tag (tag object) pointing to hash.
func (r *repo) annotatedTag(hash plumbing.Hash, name string) {
	r.t.Helper()
	_, err := r.r.CreateTag(name, hash, &gogit.CreateTagOptions{
		Message: name,
		Tagger: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  r.base,
		},
	})
	require.NoError(r.t, err)
}

func TestReadInfo(t *testing.T) {
	t.Parallel()

	t.Run("no tags, counts all commits", func(t *testing.T) {
		t.Parallel()
		r := newRepo(t)
		r.commit("first")
		r.commit("second")
		r.commit("third")

		info, err := git.ReadInfo(r.dir)

		require.NoError(t, err)
		assert.Empty(t, info.LatestTag)
		assert.Equal(t, 3, info.CommitCount)
	})

	t.Run("tag at HEAD returns zero commit count", func(t *testing.T) {
		t.Parallel()
		r := newRepo(t)
		h := r.commit("release commit")
		r.tag(h, "v1.2.3")

		info, err := git.ReadInfo(r.dir)

		require.NoError(t, err)
		assert.Equal(t, "v1.2.3", info.LatestTag)
		assert.Equal(t, semver.Version{Major: 1, Minor: 2, Patch: 3}, info.Base)
		assert.Equal(t, 0, info.CommitCount)
	})

	t.Run("counts commits since tag", func(t *testing.T) {
		t.Parallel()
		r := newRepo(t)
		h := r.commit("release")
		r.tag(h, "v2.0.0")
		r.commit("feat: one")
		r.commit("feat: two")
		r.commit("feat: three")

		info, err := git.ReadInfo(r.dir)

		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", info.LatestTag)
		assert.Equal(t, 3, info.CommitCount)
	})

	t.Run("picks most recent ancestor tag", func(t *testing.T) {
		t.Parallel()
		r := newRepo(t)
		h1 := r.commit("v1 release")
		r.tag(h1, "v1.0.0")
		h2 := r.commit("v2 release")
		r.tag(h2, "v2.0.0")
		r.commit("work after v2")

		info, err := git.ReadInfo(r.dir)

		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", info.LatestTag)
		assert.Equal(t, 1, info.CommitCount)
	})

	t.Run("non-semver tags are ignored", func(t *testing.T) {
		t.Parallel()
		r := newRepo(t)
		h := r.commit("base")
		r.tag(h, "release-2024") // ignored: not semver
		r.tag(h, "stable")       // ignored: not semver
		r.tag(h, "v1.0.0")       // counted
		r.commit("one after tag")

		info, err := git.ReadInfo(r.dir)

		require.NoError(t, err)
		assert.Equal(t, "v1.0.0", info.LatestTag)
		assert.Equal(t, 1, info.CommitCount)
	})

	t.Run("annotated tag is resolved to target commit", func(t *testing.T) {
		t.Parallel()
		r := newRepo(t)
		h := r.commit("tagged commit")
		r.annotatedTag(h, "v3.0.0")
		r.commit("after tag")
		r.commit("second after tag")

		info, err := git.ReadInfo(r.dir)

		require.NoError(t, err)
		assert.Equal(t, "v3.0.0", info.LatestTag)
		assert.Equal(t, 2, info.CommitCount)
	})

	t.Run("single commit no tag", func(t *testing.T) {
		t.Parallel()
		r := newRepo(t)
		r.commit("initial commit")

		info, err := git.ReadInfo(r.dir)

		require.NoError(t, err)
		assert.Empty(t, info.LatestTag)
		assert.Equal(t, 1, info.CommitCount)
	})

	t.Run("detects repo from subdirectory", func(t *testing.T) {
		t.Parallel()
		r := newRepo(t)
		r.commit("initial")

		subdir := filepath.Join(r.dir, "nested", "deep")
		require.NoError(t, os.MkdirAll(subdir, 0o750))

		info, err := git.ReadInfo(subdir)

		require.NoError(t, err)
		assert.Equal(t, 1, info.CommitCount)
	})

	t.Run("not a git repository returns error", func(t *testing.T) {
		t.Parallel()
		_, err := git.ReadInfo(t.TempDir())
		require.Error(t, err)
	})

	t.Run("empty repository returns error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := gogit.PlainInit(dir, false)
		require.NoError(t, err)

		_, err = git.ReadInfo(dir)
		require.Error(t, err) // no HEAD yet
	})
}
