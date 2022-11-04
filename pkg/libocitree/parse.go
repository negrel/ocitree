package libocitree

import (
	"errors"
	"fmt"

	"github.com/containers/image/v5/docker/reference"
)

var (
	ErrRepoNameContainsTagOrDigest        = errors.New("repository name can't contain any tag or digest")
	ErrRepoReferenceIsNotNamed            = errors.New("repository reference is not a named reference")
	ErrRemoteRepoReferenceContainsHeadTag = errors.New("remote repository reference contains a HEAD tag")
)

// ParseRepoReference parses the given repository reference and returns it if valid.
func ParseRepoReference(ref string) (reference.Named, error) {
	repoRef, err := parseRef(ref)
	if err != nil {
		return nil, err
	}

	if named, isNamed := repoRef.(reference.Named); isNamed {
		return named, nil
	}

	return nil, ErrRepoReferenceIsNotNamed
}

// ParseRepoName parses the given repository name and returns if it is a valid name.
// An error is returned if the reference is tagged.
func ParseRepoName(repoName string) (reference.Named, error) {
	repoRef, err := parseRepoName(repoName)
	if err != nil {
		return nil, err
	}

	repoNamedRef, isNamed := repoRef.(reference.Named)
	if !isNamed {
		return nil, ErrRepoReferenceIsNotNamed
	}

	if err := validRepoName(repoNamedRef); err != nil {
		return nil, err
	}

	return repoNamedRef, nil
}

// ParseRemoteRepoReference parses the given remote repository reference and returns it if valid.
// A remote repository reference is invalid if it contains a HEAD tag.
func ParseRemoteRepoReference(ref string) (reference.Named, error) {
	repoRef, err := ParseRepoReference(ref)
	if err != nil {
		return nil, err
	}

	if err := validRemoteRepoReference(repoRef); err != nil {
		return nil, err
	}

	return repoRef, nil
}

func parseRef(refStr string) (reference.Named, error) {
	ref, err := reference.ParseNormalizedNamed(refStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository reference: %w", err)
	}

	return ref, nil
}

func parseRepoName(repoName string) (reference.Reference, error) {
	ref, err := reference.ParseAnyReference(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository name: %w", err)
	}

	return ref, nil
}

func validRepoName(named reference.Named) error {
	if !reference.IsNameOnly(named) {
		return ErrRepoNameContainsTagOrDigest
	}

	return nil
}

func validRemoteRepoReference(remoteRef reference.Named) error {
	if tagged, isTagged := remoteRef.(reference.Tagged); isTagged && tagged.Tag() == "HEAD" {
		return ErrRemoteRepoReferenceContainsHeadTag
	}

	return nil
}
