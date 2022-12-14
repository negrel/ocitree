package reference

import (
	"errors"

	"github.com/negrel/ocitree/pkg/reference/components"
)

var (
	ErrReferenceMissingName = errors.New("reference has no name")
)

var _ Reference = LocalRepository{}
var _ Named = LocalRepository{}

// LocalRepository define an absolute reference to a local repository.
// Any named docker reference is a valid local repository reference.
// LocalRepository default tag is HEAD.
type LocalRepository struct {
	innerRef
}

// LocalFromString returns a local repository reference from the given string
// after validating and normalizing it. An error is returned if the reference is invalid.
func LocalFromString(localRef string) (LocalRepository, error) {
	name, idtag := splitComponents(localRef)
	if name == "" {
		return LocalRepository{}, ErrReferenceMissingName
	}
	if idtag == "" {
		idtag = components.Head
	}

	ref, err := newInnerRef(name, idtag)
	if err != nil {
		return LocalRepository{}, err
	}

	return LocalRepository{
		innerRef: ref,
	}, nil
}

// LocalFromRemote converts a RemoteRepository reference to a LocalRepository.
func LocalFromRemote(remoteRef RemoteRepository) LocalRepository {
	return LocalRepository{innerRef: remoteRef.innerRef}
}

// LocalHeadFromNamed returns a new local referenece with "HEAD" tag and
// name of the given named.
func LocalHeadFromNamed(ref components.Named) LocalRepository {
	l, _ := LocalFromString(ref.Name() + ":" + components.Head)
	return l
}

// LocalRebaseHeadFromNamed returns a new local reference with "REBASE_HEAD" tag and
// name of the given named.
func LocalRebaseFromNamed(ref components.Named) LocalRepository {
	l, _ := LocalFromString(ref.Name() + ":" + components.RebaseHead)
	return l
}

// LocalFromNamedTagged returns a new local reference with the given tag and
// name.
func LocalFromNamedTagged(name components.Named, tag components.Tagged) LocalRepository {
	l, _ := LocalFromString(name.Name() + ":" + tag.Tag())
	return l
}

// LocalFromNamedAndId returns a new local reference with the given
// id and name.
func LocalFromNamedAndId(name components.Named, id components.Identifier) LocalRepository {
	l, _ := LocalFromString(name.Name() + "@sha256:" + id.ID())
	return l
}

// Name implements Named interface.
func (lr LocalRepository) Name() string {
	return lr.innerRef.name.Name()
}

// IdOrTag returns either the ID or the tag of the reference.
func (lr LocalRepository) IdOrTag() string {
	return lr.innerRef.idtag.IdOrTag()
}

// NameComponent returns the name components of the reference.
func (lr LocalRepository) NameComponent() components.Name {
	return lr.innerRef.name
}

// TagComponent returns the tag components of the reference.
// Tag may be nil if reference does't contain a tag.
func (lr LocalRepository) TagComponent() *components.Tag {
	if cTag, isTag := lr.innerRef.idtag.(*components.Tag); isTag {
		return cTag
	}

	return nil
}

// IdComponent returns the identifier components of the reference.
// Id may be nil if reference does't contain a id.
func (lr LocalRepository) IdComponent() *components.ID {
	if cId, isId := lr.innerRef.idtag.(*components.ID); isId {
		return cId
	}

	return nil
}
