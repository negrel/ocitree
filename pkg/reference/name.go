package reference

import (
	"errors"
	"fmt"

	"github.com/containers/image/v5/docker/reference"
)

var (
	ErrNameInvalidFormat       = fmt.Errorf("invalid name format: %w", reference.ErrReferenceInvalidFormat)
	ErrNameContainsTagOrDigest = errors.New("name contain tag or digest")
)

var _ Named = Name{}

// Name define the name component of a repository reference.
type Name struct {
	name string
}

// NameFromString returns a Name from the given string after validating
// and normalizing it.
func NameFromString(name string) (Name, error) {
	ref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return Name{}, wrapParseError(repositoryNameParseErrorType, err)
	}

	named, isNamed := ref.(Named)
	if !isNamed {
		return Name{}, ErrNameInvalidFormat
	}

	if !reference.IsNameOnly(named) {
		return Name{}, ErrNameContainsTagOrDigest
	}

	return Name{name: named.Name()}, nil
}

func NameFromNamed(ref Named) Name {
	return Name{name: ref.Name()}
}

// String implements fmt.Stringer
func (n Name) String() string {
	return n.name
}

// Name implements Named
func (n Name) Name() string {
	return n.name
}
