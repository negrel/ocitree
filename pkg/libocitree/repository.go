package libocitree

import (
	"context"
	"errors"
	"fmt"

	"github.com/containers/common/libimage"
	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/negrel/ocitree/pkg/reference"
)

var (
	ErrRepositoryInvalidNoName = errors.New("invalid repository, no valid name")
)

type imageStore interface {
	lookupImage(reference.LocalRepository) (*libimage.Image, error)
}

// Repository is an object holding the history of a rootfs (OCI/Docker image).
type Repository struct {
	name  string
	store imageStore
	head  *libimage.Image
}

func newRepositoryFromImage(store imageStore, head *libimage.Image) (*Repository, error) {
	names, err := head.NamedTaggedRepoTags()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve image references of repository: %w", err)
	}

	repoName := findRepoName(names)
	if repoName == "" {
		return nil, ErrRepositoryInvalidNoName
	}

	return &Repository{
		name:  repoName,
		store: store,
		head:  head,
	}, nil
}

func newRepositoryFromName(store imageStore, name reference.Named) (*Repository, error) {
	ref := reference.LocalHeadFromNamed(name)

	head, err := store.lookupImage(ref)
	if err != nil {
		return nil, err
	}

	return newRepositoryFromImage(store, head)
}

// ID returns the ID of the image.
func (r *Repository) ID() string {
	return r.head.ID()
}

// Name returns the name of the repository.
func (r *Repository) Name() string {
	return r.name
}

// Tags returns other tags pointing to the same commit as HEAD.
func (r *Repository) Tags() ([]string, error) {
	names, err := r.head.NamedRepoTags()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve repository names: %w", err)
	}

	tags := make([]string, 0)

	for _, name := range names {
		if name.Name() == r.name {
			if tagged, isTagged := name.(dockerref.Tagged); isTagged {
				if t := tagged.Tag(); t != reference.HeadTag {
					tags = append(tags, t)
				}
			}
		}
	}

	return tags, nil
}

// Commits returns the commits history of this repository.
func (r *Repository) Commits() ([]Commit, error) {
	history, err := r.head.History(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve history from image: %w", err)
	}

	commits := make([]Commit, len(history))
	for i, h := range history {
		commits[i] = newCommit(h)
	}

	return commits, nil
}

// Mount mounts the repository and returns the mountpoint.
func (r *Repository) Mount() (string, error) {
	return r.head.Mount(context.Background(), []string{}, "")
}

// Unmount unmount the repository.
func (r *Repository) Unmount() error {
	return r.head.Unmount(true)
}

func findRepoName(names []reference.NamedTagged) string {
	for _, name := range names {
		if name.Tag() == reference.HeadTag {
			return name.Name()
		}
	}

	return ""
}
