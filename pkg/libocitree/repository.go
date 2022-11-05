package libocitree

import (
	"context"
	"errors"
	"fmt"

	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/docker/reference"
)

var (
	errRepositoryCorruptedNoName = errors.New("corrupted repository, no valid name")
)

type Repository struct {
	image *libimage.Image
}

func newRepository(image *libimage.Image) *Repository {
	return &Repository{
		image: image,
	}
}

// ID returns the ID of the image.
func (r *Repository) ID() string {
	return r.image.ID()
}

// Name returns the name of the repository.
func (r *Repository) Name() (string, error) {
	names, err := r.image.NamedRepoTags()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve repository names: %w", err)
	}

	return findRepoName(names)
}

// Tags returns other tags pointing to the same commit as HEAD.
func (r *Repository) Tags() ([]string, error) {
	names, err := r.image.NamedRepoTags()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve repository names: %w", err)
	}

	repoName, err := findRepoName(names)
	if err != nil {
		return nil, err
	}

	tags := make([]string, 0)

	for _, name := range names {
		if name.Name() == repoName {
			if tagged, isTagged := name.(reference.Tagged); isTagged {
				if t := tagged.Tag(); t != HeadTag {
					tags = append(tags, t)
				}
			}
		}
	}

	return tags, nil
}

func findRepoName(names []reference.Named) (string, error) {
	for _, name := range names {
		if tagged, isTagged := name.(reference.NamedTagged); isTagged {
			if tagged.Tag() == "HEAD" {
				return name.Name(), nil
			}
		}
	}

	return "", errRepositoryCorruptedNoName
}

// Commits returns the commits history of this repository.
func (r *Repository) Commits() ([]Commit, error) {
	history, err := r.image.History(context.Background())
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
	return r.image.Mount(context.Background(), []string{}, "")
}

// Unmount unmount the repository.
func (r *Repository) Unmount() (error) {
	return r.image.Unmount(true)
}
