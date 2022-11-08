package reference

import (
	"errors"

	"github.com/containers/image/v5/docker/reference"
)

var (
	ErrRemoteRepoReferenceContainsHeadTag = errors.New("remote repository reference contains a HEAD tag")
)

var (
	LatestTag = "latest"
)

var _ NamedTagged = RemoteRepository{}

// RemoteRepository is a wrapper around docker reference ensuring
// the reference doesn't contain a HEAD tag or a relative reference.
type RemoteRepository struct {
	named NamedTagged
}

// RemoteFromString returns a RemoteRepository reference from the given string
// after validating and normalizing it. An error is returned if the reference is invalid.
func RemoteFromString(remoteRef string) (RemoteRepository, error) {
	ref, err := parseRef(remoteRef)
	if err != nil {
		return RemoteRepository{}, err
	}

	namedTagged, isTagged := ref.(NamedTagged)
	if isTagged && namedTagged.Tag() == "HEAD" {
		return RemoteRepository{}, ErrRemoteRepoReferenceContainsHeadTag
	}

	if !isTagged {
		namedTagged, _ = reference.WithTag(ref, LatestTag)
	}

	return RemoteRepository{named: namedTagged}, nil
}

// RemoteLatestFromNamed returns a new RemoteReference with a "latest" tag and
// name from the given Named.
func RemoteLatestFromNamed(named Named) RemoteRepository {
	r, _ := RemoteFromString(named.Name() + ":" + LatestTag)
	return r
}

// Name implements Named
func (rr RemoteRepository) Name() string {
	return rr.named.Name()
}

// String implements reference.Reference
func (rr RemoteRepository) String() string {
	return rr.named.String()
}

// Tag implements NamedTagged
func (rr RemoteRepository) Tag() string {
	return rr.named.Tag()
}
