package reference

import (
	"errors"

	"github.com/containers/image/v5/docker/reference"
)

var (
	ErrRemoteRepoReferenceContainsReservedTag = errors.New("remote repository reference contains a reserved tag")
)

var (
	Latest = "latest"
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
	ref, err := reference.ParseNormalizedNamed(remoteRef)
	if err != nil {
		return RemoteRepository{}, wrapParseError(remoteRepositoryParseErrorType, err)
	}

	namedTagged, isTagged := ref.(NamedTagged)
	if isTagged {
		if namedTagged.Tag() == Head || namedTagged.Tag() == RebaseHead {
			return RemoteRepository{}, ErrRemoteRepoReferenceContainsReservedTag
		}
	}

	if !isTagged {
		namedTagged, _ = reference.WithTag(ref, Latest)
	}

	return RemoteRepository{named: namedTagged}, nil
}

// RemoteLatestFromNamed returns a new RemoteReference with a "latest" tag and
// name from the given Named.
func RemoteLatestFromNamed(named Named) RemoteRepository {
	r, _ := RemoteFromString(named.Name() + ":" + Latest)
	return r
}

// RemoteFromNamedTagged returns a new RemoteReference with the given
// tag and name. An error is returned if tagged is HEAD.
func RemoteFromNamedTagged(named Named, tagged Tagged) (RemoteRepository, error) {
	if tagged.Tag() == Head {
		return RemoteRepository{}, ErrRemoteRepoReferenceContainsReservedTag
	}

	return RemoteFromString(named.Name() + ":" + tagged.Tag())
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
