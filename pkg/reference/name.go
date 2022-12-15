package reference

import (
	"errors"

	"github.com/containers/image/v5/docker/reference"
)

var (
	ErrNotAName            = errors.New("not a name")
	ErrNameContainsTagOrID = errors.New("name contains tag or id")
)

// Name define the name component of a Reference.
type Name struct {
	name string
}

// Name returns a Name object from the given string after ensuring it is a valid
// name. An error is returned if the string is invalid or it contains a tag, an ID.
func NameFromString(name string) (Name, error) {
	ref, err := reference.ParseAnyReference(name)
	if err != nil {
		return Name{}, err
	}

	named, isNamed := ref.(reference.Named)
	if !isNamed {
		return Name{}, ErrNotAName
	}

	if !reference.IsNameOnly(named) {
		return Name{}, ErrNameContainsTagOrID
	}

	return Name{ref.String()}, nil
}

// NameFromNamed returns a Name object from the given named reference.
func NameFromNamed(named reference.Named) Name {
	return Name{named.Name()}
}

// String implements fmt.Stringer
func (n Name) String() string {
	return n.name
}
