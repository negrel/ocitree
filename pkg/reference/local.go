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
	name, tag, id, err := splitComponents(localRef)
	if err != nil {
		return LocalRepository{}, err
	}
	if name == "" {
		return LocalRepository{}, ErrReferenceMissingName
	}
	if tag == "" && id == "" {
		tag = components.Head
	}

	ref, err := newInnerRef(name, tag, id)
	if err != nil {
		return LocalRepository{}, err
	}

	return LocalRepository{
		innerRef: ref,
	}, nil
}

// LocalFromRemote converts a RemoteRepository reference to a LocalRepository.
//func LocalFromRemote(remoteRef RemoteRepository) LocalRepository {
//	return LocalRepository{named: remoteRef.named}
//}

// LocalHeadFromNamed returns a new LocalRepository with "HEAD" tag and
// name of the given named.
//func LocalHeadFromNamed(ref Named) LocalRepository {
//	l, _ := LocalFromString(ref.Name() + ":" + Head)
//	return l
//}

// LocalRebaseHeadFromNamed returns a new LocalRepository with "REBASE_HEAD" tag and
// name of the given named.
//func LocalRebaseFromNamed(ref Named) LocalRepository {
//	l, _ := LocalFromString(ref.Name() + ":" + RebaseHead)
//	return l
//}

// LocalFromNamedTagged returns a new LocalRepositry with the given tag and
// name.
//func LocalFromNamedTagged(named Named, tagged Tagged) LocalRepository {
//	l, _ := LocalFromString(named.Name() + ":" + tagged.Tag())
//	return l
//}

// Name implements Named interface.
func (lr LocalRepository) Name() string {
	return lr.innerRef.name.Name()
}
