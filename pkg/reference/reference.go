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

	ID() string
}

// NamedWithIdentifier is a reference which has a name and an ID.
type NamedWithIdentifier interface {
	Named
	Identifier
}

// Reference is an opaque object reference identifier that may include modifiers
// such as a local, remote and relative reference.
type Reference interface {
	AbsoluteReference() string
}

func splitComponents(ref string) (name, tag, id string, err error) {
	splitted := strings.SplitN(ref, ":", 2)
	hasTag := len(splitted) == 2
	splitted2 := strings.SplitN(ref, "@", 2)
	hasID := len(splitted2) == 2

	name = splitted[0]
	if hasTag {
		tag = splitted[1]
		if len(tag) == 0 {
			err = ErrTagInvalidFormat
			return
		}
	}
	if hasID {
		id = splitted2[1]
		if len(id) == 0 {
			err = ErrIdInvalidFormat
			return
		}
	}

	return
}

type innerRef struct {
	name components.Name
	tag  components.Tag
	id   components.ID
}

func newInnerRef(name, tag, id string) (ref innerRef, err error) {
	ref.name, err = components.NameFromString(name)
	if err != nil {
		return
	}

	if tag != "" {
		ref.tag, err = components.TagFromString(tag)
		if err != nil {
			return
		}
	}

	if id != "" {
		ref.id, err = components.IdFromString(id)
		if err != nil {
			return
		}
	}

	return
}

// AbsoluteReference implements Reference.
func (ir innerRef) AbsoluteReference() string {
	result := strings.Builder{}
	name := ir.name.Name()
	tag := ir.tag.Tag()
	id := ir.id.ID()
	result.Grow(len(name) + len(tag) + len(id) + len(":@sha256:"))

	result.WriteString(name)
	if len(tag) != 0 {
		result.WriteRune(':')
		result.WriteString(tag)
	}
	if len(id) != 0 {
		result.WriteString("@sha256:")
		result.WriteString(id)
	}

	return result.String()
}
