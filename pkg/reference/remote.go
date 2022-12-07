package reference

import (
	"errors"

	"github.com/negrel/ocitree/pkg/reference/components"
)

var (
	ErrRemoteRepoReferenceContainsReservedTag = errors.New("remote repository reference contains a reserved tag")

	reservedTags map[string]struct{} = map[string]struct{}{
		components.Head:       {},
		components.RebaseHead: {},
	}
)

var _ Reference = RemoteRepository{}
var _ Named = RemoteRepository{}

// RemoteRepository is a wrapper around docker reference ensuring
// the reference doesn't contain a HEAD tag or a relative reference.
type RemoteRepository struct {
	innerRef
}

// RemoteFromString returns a RemoteRepository reference from the given string
// after validating and normalizing it. An error is returned if the reference is invalid.
func RemoteFromString(remoteRef string) (RemoteRepository, error) {
	name, tag, id, err := splitComponents(remoteRef)
	if err != nil {
		return RemoteRepository{}, err
	}
	if name == "" {
		return RemoteRepository{}, ErrReferenceMissingName
	}
	if tag == "" && id == "" {
		tag = components.Latest
	}
	ref, err := newInnerRef(name, tag, id)
	if err != nil {
		return RemoteRepository{}, err
	}
	if _, isReserved := reservedTags[tag]; isReserved {
		return RemoteRepository{}, ErrRemoteRepoReferenceContainsReservedTag
	}

	return RemoteRepository{innerRef: ref}, nil
}

// RemoteLatestFromNamed returns a new RemoteReference with a "latest" tag and
// name from the given Named.
func RemoteLatestFromNamed(named Named) RemoteRepository {
	r, _ := RemoteFromString(named.Name() + ":" + components.Latest)
	return r
}

// RemoteFromNamedTagged returns a new RemoteReference with the given
// tag and name. An error is returned if tagged is HEAD.
func RemoteFromNamedTagged(named Named, tagged Tagged) (RemoteRepository, error) {
	if tagged.Tag() == components.Head {
		return RemoteRepository{}, ErrRemoteRepoReferenceContainsReservedTag
	}

	return RemoteFromString(named.Name() + ":" + tagged.Tag())
}

// Name implements Named.
func (rr RemoteRepository) Name() string {
	return rr.innerRef.name.Name()
}

// String implements reference.Reference
func (rr RemoteRepository) String() string {
	return rr.innerRef.AbsoluteReference()
}

// AbsoluteReference implements Reference
func (rr RemoteRepository) AbsoluteReference() string {
	return rr.String()
}
