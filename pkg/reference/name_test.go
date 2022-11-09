package reference

import (
	"testing"

	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	for _, test := range []struct {
		name          string
		refName       string
		expectedName  string
		expectedError error
	}{
		{
			name:          "EmptyName",
			refName:       "",
			expectedName:  "",
			expectedError: wrapParseError(repositoryNameParseErrorType, reference.ErrReferenceInvalidFormat),
		},
		{
			name:         "Minimal/Valid",
			refName:      "archlinux",
			expectedName: "docker.io/library/archlinux",
		},
		{
			name:         "FullyQualified/Valid",
			refName:      "docker.io/library/archlinux",
			expectedName: "docker.io/library/archlinux",
		},
		{
			name:          "WithTags/Invalid",
			refName:       "docker.io/library/archlinux:latest",
			expectedError: ErrNameContainsTagOrDigest,
		},
		{
			name:          "WithDigest/Invalid",
			refName:       "docker.io/library/archlinux@sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf",
			expectedError: ErrNameContainsTagOrDigest,
		},
		{
			name:          "InvalidName",
			refName:       "§archlinux",
			expectedError: wrapParseError(repositoryNameParseErrorType, reference.ErrReferenceInvalidFormat),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ref, err := NameFromString(test.refName)
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedName, ref.String())
		})
	}
}
