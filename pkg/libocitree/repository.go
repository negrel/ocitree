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

