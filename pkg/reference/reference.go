package reference

import (
	"errors"
	"strings"

	"github.com/containers/image/v5/docker/reference"
	"github.com/negrel/ocitree/pkg/reference/components"
)

var (
	ErrReferenceInvalidFormat = reference.ErrReferenceInvalidFormat
	ErrTagInvalidFormat       = reference.ErrTagInvalidFormat
	ErrIdInvalidFormat        = errors.New("invalid id format")

	ErrReferenceMissingIdOrTag = errors.New("reference has no tag or id")
)

// Named is a reference with a name.
type Named interface {
	Reference
	components.Named
}

// NamedTagged is a reference including a name and tag.
type NamedTagged interface {
	Named
	Tagged
}

// Tagged is a reference which has a tag.
type Tagged interface {
	Reference
	components.Tagged
}

// Identifier is a reference which has an ID.
type Identifier interface {
	Reference
	components.Identifier
}

// NamedWithIdentifier is a reference which has a name and an ID.
type NamedWithIdentifier interface {
	Named
	Identifier
}

// EitherTaggedOrIdentifier define a reference which has a name
// and either an ID or a Tag.
type EitherTaggedOrIdentifier interface {
	Named
	IdOrTag() string
}

// Reference is an opaque object reference identifier that may include modifiers
// such as a local, remote and relative reference.
type Reference interface {
	AbsoluteReference() string
}

func splitComponents(ref string) (name, idtag string) {
	splitted := strings.SplitN(ref, components.IdPrefix, 2)
	if hasID := len(splitted) == 2; hasID {
		return splitted[0], splitted[1]
	}

	splitted = strings.SplitN(ref, components.TagPrefix, 2)
	if hasTag := len(splitted) == 2; hasTag {
		return splitted[0], splitted[1]
	}

	return ref, ""
}

type innerRef struct {
	name  components.Name
	idtag components.IdentifierOrTag
}

func newInnerRef(name, idtag string) (ref innerRef, err error) {
	ref.name, err = components.NameFromString(name)
	if err != nil {
		return
	}

	ref.idtag, err = components.IdentifierOrTagFromString(idtag)
	if err != nil {
		return
	}

	return
}

// AbsoluteReference implements Reference.
func (ir innerRef) AbsoluteReference() string {
	result := strings.Builder{}
	name := ir.name.Name()
	idtag := ir.idtag.IdOrTag()
	result.Grow(len(name) + len(idtag))

	result.WriteString(name)
	result.WriteString(idtag)

	return result.String()
}

type dockerRef struct {
	Reference
}

// Name implements reference.Named
func (dr dockerRef) Name() string {
	return dr.Reference.(Named).Name()
}

// String implements reference.Reference
func (dr dockerRef) String() string {
	return dr.AbsoluteReference()
}

// DockerRef wraps an absolute reference and converts it to reference.Reference.
func DockerRef(ref Reference) reference.Reference {
	return dockerRef{Reference: ref}
}

// DockerRef wraps an absolute named reference and converts it to reference.Named.
func NamedDockerRef(ref Named) reference.Named {
	return dockerRef{Reference: ref}
}
