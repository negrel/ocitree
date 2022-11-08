package reference

import "github.com/containers/image/v5/docker/reference"

const (
	HeadTag = "HEAD"
)

var _ NamedTagged = LocalRepository{}

// LocalRepository define an absolute reference to a local repository.
// Any named docker reference is a valid local repository reference.
// LocalRepository default tag is HEAD.
type LocalRepository struct {
	named NamedTagged
}

// LocalFromString returns a local repository reference from the given string
// after validating and normalizing it. An error is returned if the reference is invalid.
func LocalFromString(localRef string) (LocalRepository, error) {
	ref, err := parseRef(localRef)
	if err != nil {
		return LocalRepository{}, err
	}

	namedTagged, isTagged := ref.(NamedTagged)
	if !isTagged {
		namedTagged, _ = reference.WithTag(ref, HeadTag)
	}

	return LocalRepository{named: namedTagged}, nil
}

// LocalFromRemote converts a RemoteRepository reference to a LocalRepository.
func LocalFromRemote(remoteRef RemoteRepository) LocalRepository {
	return LocalRepository{named: remoteRef.named}
}

// LocalHeadFromNamed returns a new LocalRepository with "HEAD" tag and
// name of the given named.
func LocalHeadFromNamed(ref Named) LocalRepository {
	l, _ := LocalFromString(ref.Name() + ":" + HeadTag)
	return l
}

// String implements reference.Reference
func (lr LocalRepository) String() string {
	return lr.named.String()
}

// Name implements Named
func (lr LocalRepository) Name() string {
	return lr.named.Name()
}

// Tag implements NamedTagged
func (lr LocalRepository) Tag() string {
	return lr.named.Tag()
}