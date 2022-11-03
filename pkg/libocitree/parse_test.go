package libocitree

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRemoteRepoReference(t *testing.T) {
	errInvalidReferenceFormat := fmt.Errorf("failed to parse image reference: invalid reference format")

	for _, test := range []struct {
		ref           string
		expectedError error
	}{
		{"docker.io/library/ubuntu", nil},
		{"docker.io/library/ubuntu:latest", nil},
		{"docker.io/library/ubuntu:22.04", nil},
		{"docker.io/library/ubuntu:22.04@sha256:a428de44a9059f31a59237a5881c2d2cffa93757d99026156e4ea544577ab7f3", nil},
		{"docker.io/library/ubuntu@sha256:a428de44a9059f31a59237a5881c2d2cffa93757d99026156e4ea544577ab7f3", nil},
		{"library/ubuntu", nil},
		{"library/ubuntu:latest", nil},
		{"library/ubuntu:22.04", nil},
		{"ubuntu", nil},
		{"ubuntu:latest", nil},
		{"ubuntu:22.04", nil},

		{"", errInvalidReferenceFormat},
		{"docker.io/library/ubuntu:HEAD", ErrRemoteRepoReferenceContainsHeadTag},
		{"§§§", errInvalidReferenceFormat},
		{"§§§/ubuntu", errInvalidReferenceFormat},
		{"§§§/ubuntu:latest", errInvalidReferenceFormat},
		{"§§§/ubuntu:22.04", errInvalidReferenceFormat},
		{"/ubuntu", errInvalidReferenceFormat},
		{"/ubuntu:latest", errInvalidReferenceFormat},
		{"/ubuntu:22.04", errInvalidReferenceFormat},
	} {
		t.Run(test.ref, func(t *testing.T) {
			_, err := ParseRemoteRepoReference(test.ref)
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError.Error(), err.Error(), "error message doesn't match expected one")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseRepoReference(t *testing.T) {
	errInvalidReferenceFormat := fmt.Errorf("failed to parse image reference: invalid reference format")

	for _, test := range []struct {
		ref           string
		expectedError error
	}{
		{"docker.io/library/ubuntu", nil},
		{"docker.io/library/ubuntu:latest", nil},
		{"docker.io/library/ubuntu:22.04", nil},
		{"docker.io/library/ubuntu:22.04@sha256:a428de44a9059f31a59237a5881c2d2cffa93757d99026156e4ea544577ab7f3", nil},
		{"docker.io/library/ubuntu@sha256:a428de44a9059f31a59237a5881c2d2cffa93757d99026156e4ea544577ab7f3", nil},
		{"library/ubuntu", nil},
		{"library/ubuntu:latest", nil},
		{"library/ubuntu:22.04", nil},
		{"ubuntu", nil},
		{"ubuntu:latest", nil},
		{"ubuntu:22.04", nil},
		{"docker.io/library/ubuntu:HEAD", nil},

		{"", errInvalidReferenceFormat},
		{"§§§", errInvalidReferenceFormat},
		{"§§§/ubuntu", errInvalidReferenceFormat},
		{"§§§/ubuntu:latest", errInvalidReferenceFormat},
		{"§§§/ubuntu:22.04", errInvalidReferenceFormat},
		{"/ubuntu", errInvalidReferenceFormat},
		{"/ubuntu:latest", errInvalidReferenceFormat},
		{"/ubuntu:22.04", errInvalidReferenceFormat},
	} {
		t.Run(test.ref, func(t *testing.T) {
			_, err := ParseRepoReference(test.ref)
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError.Error(), err.Error(), "error message doesn't match expected one")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseRepoName(t *testing.T) {
	errInvalidReferenceFormat := fmt.Errorf("failed to parse image reference: invalid reference format")

	for _, test := range []struct {
		ref           string
		expectedError error
	}{
		{"docker.io/library/ubuntu", nil},
		{"library/ubuntu", nil},
		{"ubuntu", nil},

		{"", errInvalidReferenceFormat},
		{"§§§", errInvalidReferenceFormat},
		{"§§§/ubuntu", errInvalidReferenceFormat},
		{"§§§/ubuntu:latest", errInvalidReferenceFormat},
		{"§§§/ubuntu:22.04", errInvalidReferenceFormat},
		{"/ubuntu", errInvalidReferenceFormat},
		{"/ubuntu:latest", errInvalidReferenceFormat},
		{"/ubuntu:22.04", errInvalidReferenceFormat},
	
		{"docker.io/library/ubuntu:latest", ErrRepoNameContainsTag},
		{"docker.io/library/ubuntu:22.04", ErrRepoNameContainsTag},
		{"docker.io/library/ubuntu:22.04@sha256:a428de44a9059f31a59237a5881c2d2cffa93757d99026156e4ea544577ab7f3", ErrRepoNameContainsTag},
		{"docker.io/library/ubuntu@sha256:a428de44a9059f31a59237a5881c2d2cffa93757d99026156e4ea544577ab7f3", ErrRepoNameContainsDigest},
		{"ubuntu:latest", ErrRepoNameContainsTag},
		{"ubuntu:22.04", ErrRepoNameContainsTag},
		{"library/ubuntu:latest", ErrRepoNameContainsTag},
		{"library/ubuntu:22.04", ErrRepoNameContainsTag},
	} {
		t.Run(test.ref, func(t *testing.T) {
			_, err := ParseRepoName(test.ref)
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError.Error(), err.Error(), "error message doesn't match expected one")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
