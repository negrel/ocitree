package reference

import (
	"errors"
	"strings"

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
	name, idtag := splitComponents(remoteRef)
	if name == "" {
		return RemoteRepository{}, ErrReferenceMissingName
	}
	if idtag == "" {
		idtag = components.Latest
	}
	ref, err := newInnerRef(name, idtag)
	if err != nil {
		return RemoteRepository{}, err
	}
	if _, isReserved := reservedTags[idtag]; isReserved {
		return RemoteRepository{}, ErrRemoteRepoReferenceContainsReservedTag
	}

	return RemoteRepository{innerRef: ref}, nil
}

// RemoteLatestFromName returns a new remote reference with a "latest" tag and
// name from the given Named.
func RemoteLatestFromName(named components.Name) RemoteRepository {
	r, _ := RemoteFromString(named.Name() + ":" + components.Latest)
	return r
}

// RemoteFromNamedAndTagged returns a new remote reference with the given
// tag and name. An error is returned if the tag is a reseved tag (local reference).
func RemoteFromNamedAndIdTag(named components.Named, idtag components.IdentifierOrTag) (RemoteRepository, error) {
	tag := strings.TrimPrefix(idtag.IdOrTag(), components.TagPrefix)
	if _, isReserved := reservedTags[tag]; isReserved {
		return RemoteRepository{}, ErrRemoteRepoReferenceContainsReservedTag
	}

	return RemoteFromString(named.Name() + idtag.IdOrTag())
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

// NameComponent returns the name components of the reference.
func (rr RemoteRepository) NameComponent() *components.Name {
	return &rr.innerRef.name
}

// IdOrTag returns either the ID or the tag of the reference.
func (rr RemoteRepository) IdOrTag() string {
	return rr.innerRef.idtag.IdOrTag()
}

// TagComponent returns the tag components of the reference.
// Tag may be nil if reference does't contain a tag.
func (rr RemoteRepository) TagComponent() *components.Tag {
	if cTag, isTag := rr.innerRef.idtag.(*components.Tag); isTag {
		return cTag
	}

	return nil
}

// IdComponent returns the identifier components of the reference.
// Id may be nil if reference does't contain a id.
func (rr RemoteRepository) IdComponent() *components.ID {
	if cId, isId := rr.innerRef.idtag.(*components.ID); isId {
		return cId
	}

	return nil
}
