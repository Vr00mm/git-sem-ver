// Package git provides version-relevant metadata by reading the local
// git repository directly via go-git (no subprocess, no git CLI dependency).
package git

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Vr00mm/git-sem-ver/internal/semver"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// repository is the subset of go-git's Repository API we actually use.
// Keeping it minimal makes it straightforward to mock in tests.
type repository interface {
	Head() (*plumbing.Reference, error)
	Tags() (storer.ReferenceIter, error)
	Log(*gogit.LogOptions) (object.CommitIter, error)
	TagObject(plumbing.Hash) (*object.Tag, error)
}

// Info holds the version-relevant data extracted from the repository.
type Info struct {
	// LatestTag is the most recent semver tag reachable from HEAD (e.g. "v1.2.3"),
	// or empty if no semver tag exists yet.
	LatestTag string
	// Base is the parsed semver of LatestTag. Zero value (0.0.0) when no tag exists.
	Base semver.Version
	// CommitCount is the number of commits between LatestTag and HEAD.
	// If LatestTag is empty, it counts all commits reachable from HEAD.
	CommitCount int
}

// ReadInfo opens the repository at path (use "." for the working directory)
// and returns version-relevant metadata.
func ReadInfo(path string) (Info, error) {
	repo, err := gogit.PlainOpenWithOptions(path, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return Info{}, fmt.Errorf("opening repository at %q: %w", path, err)
	}
	return readInfo(repo)
}

// readInfo is the testable core of ReadInfo; it operates on the repository interface.
func readInfo(repo repository) (Info, error) {
	head, err := repo.Head()
	if err != nil {
		return Info{}, fmt.Errorf("reading HEAD: %w", err)
	}

	tagsByHash, err := semverTagsByHash(repo)
	if err != nil {
		return Info{}, err
	}

	return walkHistory(repo, head.Hash(), tagsByHash)
}

// semverTagsByHash returns a map of commit hash → tag name for all semver tags.
func semverTagsByHash(repo repository) (map[plumbing.Hash]semver.Version, error) {
	tags := make(map[plumbing.Hash]semver.Version)

	iter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	defer iter.Close()

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		v, ok := parseSemverTag(name)
		if !ok {
			return nil
		}

		hash := ref.Hash()

		// Resolve annotated tags to their target commit.
		if tagObj, err := repo.TagObject(hash); err == nil {
			hash = tagObj.Target
		}

		tags[hash] = v
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	return tags, nil
}

// walkHistory walks commits from HEAD backwards and counts how many commits
// have been made since the most recent semver tag ancestor.
func walkHistory(repo repository, from plumbing.Hash, tags map[plumbing.Hash]semver.Version) (Info, error) {
	iter, err := repo.Log(&gogit.LogOptions{From: from})
	if err != nil {
		return Info{}, fmt.Errorf("reading commit log: %w", err)
	}
	defer iter.Close()

	count := 0
	for {
		c, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Info{}, fmt.Errorf("walking commit history: %w", err)
		}

		if v, ok := tags[c.Hash]; ok {
			return Info{
				LatestTag:   "v" + v.String(),
				Base:        v,
				CommitCount: count,
			}, nil
		}
		count++
	}

	// No semver tag found anywhere in history.
	return Info{CommitCount: count}, nil
}

// parseSemverTag returns the parsed version and true if name is a valid semver tag.
func parseSemverTag(name string) (semver.Version, bool) {
	s := strings.TrimPrefix(name, "v")
	v, err := semver.Parse(s)
	if err != nil {
		return semver.Version{}, false
	}
	return v, true
}
