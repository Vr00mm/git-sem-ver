package git

import (
	"errors"
	"io"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/stretchr/testify/require"
)

// funcRepo is a flexible mock that delegates each method to a provided function.
type funcRepo struct {
	headFn   func() (*plumbing.Reference, error)
	tagsFn   func() (storer.ReferenceIter, error)
	logFn    func(*gogit.LogOptions) (object.CommitIter, error)
	tagObjFn func(plumbing.Hash) (*object.Tag, error)
}

func (f *funcRepo) Head() (*plumbing.Reference, error)                 { return f.headFn() }
func (f *funcRepo) Tags() (storer.ReferenceIter, error)                { return f.tagsFn() }
func (f *funcRepo) Log(o *gogit.LogOptions) (object.CommitIter, error) { return f.logFn(o) }
func (f *funcRepo) TagObject(h plumbing.Hash) (*object.Tag, error)     { return f.tagObjFn(h) }

// eofReferenceIter is an iterator that yields nothing.
type eofReferenceIter struct{}

func (eofReferenceIter) Next() (*plumbing.Reference, error)            { return nil, io.EOF }
func (eofReferenceIter) ForEach(func(*plumbing.Reference) error) error { return nil }
func (eofReferenceIter) Close()                                        {}

// errReferenceIter is an iterator whose ForEach returns an error.
type errReferenceIter struct{}

func (errReferenceIter) Next() (*plumbing.Reference, error) { return nil, io.EOF }
func (errReferenceIter) ForEach(func(*plumbing.Reference) error) error {
	return errors.New("iter error")
}
func (errReferenceIter) Close() {}

// errCommitIter is an iterator whose Next returns an error.
type errCommitIter struct{}

func (errCommitIter) Next() (*object.Commit, error)            { return nil, errors.New("commit iter error") }
func (errCommitIter) ForEach(func(*object.Commit) error) error { return nil }
func (errCommitIter) Close()                                   {}

// validHead returns a HEAD reference pointing at a non-zero hash.
func validHead() *plumbing.Reference {
	return plumbing.NewHashReference(
		plumbing.NewBranchReferenceName("main"),
		plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
	)
}

func TestReadInfo_errorPaths(t *testing.T) {
	t.Parallel()

	t.Run("Head returns error", func(t *testing.T) {
		t.Parallel()
		repo := &funcRepo{
			headFn: func() (*plumbing.Reference, error) {
				return nil, errors.New("head failed")
			},
		}
		_, err := readInfo(repo)
		require.ErrorContains(t, err, "reading HEAD")
	})

	t.Run("Tags returns error", func(t *testing.T) {
		t.Parallel()
		repo := &funcRepo{
			headFn: func() (*plumbing.Reference, error) { return validHead(), nil },
			tagsFn: func() (storer.ReferenceIter, error) {
				return nil, errors.New("tags failed")
			},
		}
		_, err := readInfo(repo)
		require.ErrorContains(t, err, "listing tags")
	})

	t.Run("Tags iterator returns error", func(t *testing.T) {
		t.Parallel()
		repo := &funcRepo{
			headFn: func() (*plumbing.Reference, error) { return validHead(), nil },
			tagsFn: func() (storer.ReferenceIter, error) { return errReferenceIter{}, nil },
		}
		_, err := readInfo(repo)
		require.ErrorContains(t, err, "iterating tags")
	})

	t.Run("Log returns error", func(t *testing.T) {
		t.Parallel()
		repo := &funcRepo{
			headFn:   func() (*plumbing.Reference, error) { return validHead(), nil },
			tagsFn:   func() (storer.ReferenceIter, error) { return eofReferenceIter{}, nil },
			tagObjFn: func(plumbing.Hash) (*object.Tag, error) { return nil, errors.New("not a tag") },
			logFn: func(*gogit.LogOptions) (object.CommitIter, error) {
				return nil, errors.New("log failed")
			},
		}
		_, err := readInfo(repo)
		require.ErrorContains(t, err, "reading commit log")
	})

	t.Run("Commit iterator returns error mid-walk", func(t *testing.T) {
		t.Parallel()
		repo := &funcRepo{
			headFn:   func() (*plumbing.Reference, error) { return validHead(), nil },
			tagsFn:   func() (storer.ReferenceIter, error) { return eofReferenceIter{}, nil },
			tagObjFn: func(plumbing.Hash) (*object.Tag, error) { return nil, errors.New("not a tag") },
			logFn:    func(*gogit.LogOptions) (object.CommitIter, error) { return errCommitIter{}, nil },
		}
		_, err := readInfo(repo)
		require.ErrorContains(t, err, "walking commit history")
	})
}
