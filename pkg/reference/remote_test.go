package reference

import (
	"testing"

	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/require"
)

func TestRemoteReference(t *testing.T) {
	for _, test := range []struct {
		name              string
		reference         string
		expectedName      string
		expectedTag       string
		expectedReference string
		expectedError     error
	}{
		{
			name:          "EmptyInvalid",
			reference:     "",
			expectedError: wrapParseError(remoteRepositoryParseErrorType, reference.ErrReferenceInvalidFormat),
		},
		{
			name:              "FullyQualified/WithCustomDomain/Valid",
			reference:         "negrel.dev/archlinux:latest",
			expectedName:      "negrel.dev/archlinux",
			expectedTag:       "latest",
			expectedReference: "negrel.dev/archlinux:latest",
		},
		{
			name:              "FullyQualified/WithCustomDomain/Valid",
			reference:         "negrel.dev/library/archlinux:latest",
			expectedName:      "negrel.dev/library/archlinux",
			expectedTag:       "latest",
			expectedReference: "negrel.dev/library/archlinux:latest",
		},
		{
			name:              "FullyQualified/WithCustomTag/Valid",
			reference:         "docker.io/library/archlinux:edge",
			expectedName:      "docker.io/library/archlinux",
			expectedTag:       "edge",
			expectedReference: "docker.io/library/archlinux:edge",
		},
		{
			name:          "FullyQualified/WithHEADTag/Invalid",
			reference:     "docker.io/library/archlinux:HEAD",
			expectedError: ErrRemoteRepoReferenceContainsReservedTag,
		},
		{
			name:          "FullyQualified/WithEmptyTag/Invalid",
			reference:     "docker.io/library/archlinux:",
			expectedError: wrapParseError(remoteRepositoryParseErrorType, reference.ErrReferenceInvalidFormat),
		},
		{
			name:              "FullyQualifiedLocalhostValid",
			reference:         "localhost/archlinux:edge",
			expectedName:      "localhost/archlinux",
			expectedTag:       "edge",
			expectedReference: "localhost/archlinux:edge",
		},
		{
			name:              "FullyQualified/Valid",
			reference:         "docker.io/library/archlinux:edge",
			expectedName:      "docker.io/library/archlinux",
			expectedTag:       "edge",
			expectedReference: "docker.io/library/archlinux:edge",
		},
		{
			name:          "FullyQualified/InvalidTag",
			reference:     "docker.io/library/archlinux:...",
			expectedError: wrapParseError(remoteRepositoryParseErrorType, reference.ErrReferenceInvalidFormat),
		},
		{
			name:          "InvalidDomain",
			reference:     ".docker.io/library/archlinux:latest",
			expectedError: wrapParseError(remoteRepositoryParseErrorType, reference.ErrReferenceInvalidFormat),
		},
		{
			name:          "InvalidName",
			reference:     "docker.io/library/§archlinux§:latest",
			expectedError: wrapParseError(remoteRepositoryParseErrorType, reference.ErrReferenceInvalidFormat),
		},
		{
			name:              "MissingDomain/Valid",
			reference:         "archlinux:latest",
			expectedName:      "docker.io/library/archlinux",
			expectedTag:       "latest",
			expectedReference: "docker.io/library/archlinux:latest",
		},
		{
			name:              "MissingDomainAndTag/Valid",
			reference:         "archlinux",
			expectedName:      "docker.io/library/archlinux",
			expectedTag:       "latest",
			expectedReference: "docker.io/library/archlinux:latest",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ref, err := RemoteFromString(test.reference)
			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedName, ref.Name())
			require.Equal(t, test.expectedReference, ref.String())
		})
	}
}
