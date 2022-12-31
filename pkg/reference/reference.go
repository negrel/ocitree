package reference

import (
	"errors"
	"fmt"
	"strings"

	"github.com/containers/image/v5/docker/reference"
	"github.com/opencontainers/go-digest"
)

const (
	// HEAD reserved tag
	Head = "HEAD"
	// REBASE_HEAD reserved tag
	RebaseHead = "REBASE_HEAD"

	Latest = "latest"

	IdPrefix  = "@sha256:"
	TagPrefix = ":"
)

var (
	ErrIDInvalidFormat = errors.New("invalid id format")
	ErrTagIsReserved   = errors.New("tag is reserved")

	reservedTags map[string]struct{} = map[string]struct{}{
		Head:       {},
		RebaseHead: {},
	}

	HeadTag       = LocalTagFromTag(tag{TagPrefix + Head})
	RebaseHeadTag = LocalTagFromTag(tag{TagPrefix + RebaseHead})
	LatestTag     = RemoteTagFromTag(tag{TagPrefix + Latest})
)

// Reference defines a repository reference.
// A reference composed of a repository name and an ID or a Tag.
//
// NAME[:TAG | @sha256:ID]
type Reference interface {
	String() string
	Name() Name
	IdOrTag
}

type IdOrTag interface {
	IdOrTag() string
}

// DockerRefFromReference returns a docker reference from the given
// repository reference.
func DockerRefFromReference(ref Reference) reference.Named {
	named, err := reference.ParseDockerRef(ref.String())
	if err != nil {
		panic(err)
	}

	return named
}

// LocalRef defines a local repository reference.
type LocalRef interface {
	Reference

	privateLocalRef()
}

func LocalRefFromString(rawRef string) (LocalRef, error) {
	ref, err := reference.ParseDockerRef(rawRef)
	if err != nil {
		return nil, err
	}

	if digested, isDigested := ref.(reference.Digested); isDigested {
		return NewLocal(NameFromNamed(ref), IDFromDigest(digested.Digest())), nil
	} else if tagged, isTagged := ref.(reference.Tagged); isTagged {
		// Overwrite latest tag if added by reference.ParseDockerRef
		if tagged.Tag() == Latest && !strings.HasSuffix(rawRef, Latest) {
			return NewLocal(NameFromNamed(ref), HeadTag), nil
		}

		return NewLocal(NameFromNamed(ref), LocalTagFromTag(tagged)), nil
	}

	return NewLocal(NameFromNamed(ref), HeadTag), nil
}

type IdOrLocalTag interface {
	ID | LocalTag
	fmt.Stringer
}

var _ LocalRef = Local[ID]{}
var _ LocalRef = Local[LocalTag]{}

// Local defines a concrete local reference with either an ID or a local tag.
type Local[T IdOrLocalTag] struct {
	ref[T]
}

// NewLocal returns a new Local reference with the given name and ID or Tag.
func NewLocal[T IdOrLocalTag](name Name, idOrTag T) Local[T] {
	return Local[T]{
		ref: newRef(name, idOrTag),
	}
}

// LocalFromName returns a new Local reference with the given name and a Head tag.
func LocalFromName(name Name) Local[LocalTag] {
	return NewLocal(name, HeadTag)
}

func (l Local[T]) privateLocalRef() {}

type RemoteRef interface {
	Reference

	privateRemoteRef()
}

func RemoteRefFromString(rawRef string) (RemoteRef, error) {
	ref, err := reference.ParseDockerRef(rawRef)
	if err != nil {
		return nil, err
	}

	if digested, isDigested := ref.(reference.Digested); isDigested {
		return NewRemote(NameFromNamed(ref), IDFromDigest(digested.Digest())), nil
	} else if tagged, isTagged := ref.(reference.Tagged); isTagged {
		tag, err := RemoteTagFromString(tagged.Tag())
		if err != nil {
			return nil, err
		}

		return NewRemote(NameFromNamed(ref), tag), nil
	}

	return NewRemote(NameFromNamed(ref), LatestTag), nil
}

type IdOrRemoteTag interface {
	ID | RemoteTag
	fmt.Stringer
}

var _ RemoteRef = Remote[ID]{}
var _ RemoteRef = Remote[RemoteTag]{}

// Remote define a concrete remote reference with either an ID or a remote tag.
type Remote[T IdOrRemoteTag] struct {
	ref[T]
}

// NewRemote returns a new Remote reference with the given name and tag.
func NewRemote[T IdOrRemoteTag](name Name, idOrTag T) Remote[T] {
	return Remote[T]{
		ref: newRef(name, idOrTag),
	}
}

// RemoteFromName returns a new Remote reference with the given name and
// a Latest tag.
func RemoteFromName(name Name) Remote[RemoteTag] {
	return NewRemote(name, LatestTag)
}

func (r Remote[T]) privateRemoteRef() {}

type ref[T IdOrTagConstraint] struct {
	name    Name
	idOrTag T
}

func newRef[T IdOrTagConstraint](name Name, idOrTag T) ref[T] {
	return ref[T]{
		name:    name,
		idOrTag: idOrTag,
	}
}

func (r ref[T]) String() string {
	return fmt.Sprintf("%v%v", r.name, r.idOrTag)
}

func (r ref[T]) Name() Name {
	return r.name
}

func (r ref[T]) IdOrTag() string {
	return r.idOrTag.String()
}

func (r ref[T]) GetIdOrTag() T {
	return r.idOrTag
}

// IdOrTagConstraint is either an ID or a Tag.
type IdOrTagConstraint interface {
	ID | TagConstraint
	fmt.Stringer
}

// TagConstraint defines either a LocalTag or a RemoteTag.
type TagConstraint interface {
	LocalTag | RemoteTag
}

// Tag define objects with a tag.
type Tag interface {
	Tag() string
}

// ID defines the ID component of a repository reference.
type ID struct {
	inner string
}

// IDFromString returns a new ID if the given string is a valid ID.
func IDFromString(id string) (ID, error) {
	if reference.IdentifierRegexp.MatchString(id) {
		return ID{IdPrefix + id}, nil
	}

	return ID{}, reference.ErrTagInvalidFormat
}

// IDFromDigested returns an ID from the given digest.
func IDFromDigest(d digest.Digest) ID {
	id := IdPrefix + d.Encoded()

	return ID{id}
}

// String implements fmt.Stringer.
func (i ID) String() string {
	return i.inner
}

// tag defines the tag component of a repository reference.
type tag struct {
	inner string
}

// Tag implements reference.Tagged
func (t tag) Tag() string {
	return t.inner[len(TagPrefix):]
}

func newTag(rawTag string) (tag, error) {
	if reference.TagRegexp.MatchString(rawTag) {
		return tag{TagPrefix + rawTag}, nil
	}

	return tag{}, reference.ErrTagInvalidFormat
}

func tagFromTag(t Tag) tag {
	return tag{TagPrefix + t.Tag()}
}

// String implements fmt.Stringer.
func (t tag) String() string {
	return t.inner
}

// LocalTag defines a tag for local reference.
// Any docker tag is a valid local tag.
type LocalTag struct {
	tag
}

// LocalTagFromString creates a new LocalTag after validating the given tag.
func LocalTagFromString(rawTag string) (LocalTag, error) {
	innerTag, err := newTag(rawTag)
	if err != nil {
		return LocalTag{}, err
	}

	return LocalTag{innerTag}, nil
}

// LocalTagFromTagged creates a new LocalTag from the given Tagged reference.
func LocalTagFromTag(tag Tag) LocalTag {
	return LocalTag{tagFromTag(tag)}
}

// RemoteTag defines a tag for remote reference.
// Remote tag can't contains reserved tag such as HEAD, REBASE_HEAD, etc.
type RemoteTag struct {
	tag
}

// RemoteTagFromString creates a new RemoteTag after validating the given tag.
func RemoteTagFromString(rawTag string) (RemoteTag, error) {
	if _, isReserved := reservedTags[rawTag]; isReserved {
		return RemoteTag{}, ErrTagIsReserved
	}

	innerTag, err := newTag(rawTag)
	if err != nil {
		return RemoteTag{}, err
	}

	return RemoteTag{innerTag}, nil
}

// RemoteTagFromTag creates a new RemoteTag from the given Tagged reference.
// This function panic if the given tag is a reserved tag.
func RemoteTagFromTag(tag Tag) RemoteTag {
	if _, isReserved := reservedTags[tag.Tag()]; isReserved {
		panic(ErrTagIsReserved)
	}

	return RemoteTag{tagFromTag(tag)}
}
