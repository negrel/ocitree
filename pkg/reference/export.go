package reference

import dockerref "github.com/containers/image/v5/docker/reference"

// Named is an object with a full name.
type Named = dockerref.Named

// NamedTagged is an object including a name and tag.
type NamedTagged = dockerref.NamedTagged

// Tagged is an object which has a tag.
type Tagged = dockerref.Tagged

// Reference is an opaque object reference identifier that may include modifiers
// such as a local, remote and relative reference.
type Reference interface {
	AbsoluteReference() string
}
