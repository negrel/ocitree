package reference

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/negrel/ocitree/pkg/reference/components"
)

var (
	ErrInvalidOffsetFormat = errors.New("invalid offset format")
)

// Relative defines a relative reference.
// It is made of a base and an offset.
// Base is the reference on which the offset must be applied to get
// an absolute reference.
type Relative struct {
	ref    Reference
	offset uint
}

// RelativeFromReferenceAndOffset returns a new Relative reference base on the given
// reference and offset.
func RelativeFromReferenceAndOffset(ref Reference, offset uint) Relative {
	return Relative{ref: ref, offset: offset}
}

var offsetRegex = regexp.MustCompile(`(~\d+|\^+)$`)

// RelativeFromString parses the given string and returns relative reference
// after validating and normalizing it. An error is returned if the reference is invalid.
func RelativeFromString(ref string) (Relative, error) {
	index := offsetRegex.FindStringIndex(ref)
	offset := uint(0)

	// Parse offset if there is one
	if index != nil {
		var err error
		offset, err = parseOffset(ref[index[0]:index[1]])
		if err != nil {
			return Relative{}, fmt.Errorf("failed to parse offset: %w", err)
		}

		// Strip offset from ref
		ref = ref[:index[0]]
		if ref[len(ref)-1] == ':' {
			ref = ref[:len(ref)-1]
		}
	}

	// Parse base reference
	var baseRef Reference
	name, idtag := splitComponents(ref)
	if idtag == "" {
		idtag = components.Head
	}

	baseRef, err := newInnerRef(name, idtag)
	if err != nil {
		return Relative{}, err
	}

	return RelativeFromReferenceAndOffset(
		baseRef,
		offset,
	), nil
}

// Base returns the base of the relative reference
func (r Relative) Base() Reference {
	return r.ref
}

// Offset returns the offset part of the relative reference.
func (r Relative) Offset() uint {
	return r.offset
}

// parseOffset returns the offset value of a relative reference string.
// NOTE: this function must not be called directly, RelativeFromString should
// be used instead.
func parseOffset(offset string) (uint, error) {
	if len(offset) == 0 {
		panic("offset string is empty")
	}

	switch offset[0] {
	case '^':
		return uint(len(offset)), nil
	case '~':
		if len(offset) == 1 {
			return 1, nil
		}
		result, err := strconv.Atoi(offset[1:])
		if err != nil {
			return 0, fmt.Errorf("failed to parse integer of relative reference: %w", err)
		}

		return uint(result), nil

	default:
		return 0, ErrInvalidOffsetFormat
	}
}
