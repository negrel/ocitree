package reference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoteReference(t *testing.T) {
	for _, test := range []struct {
		name              string
		reference         string
		expectedName      string
		expectedTag       string
		expectedReference string
		expectedErrorMsg  string
	}{
		{
			name:             "EmptyInvalid",
			reference:        "",
			expectedErrorMsg: ErrReferenceMissingName.Error(),
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
			name:             "FullyQualified/WithHEADTag/Invalid",
			reference:        "docker.io/library/archlinux:HEAD",
			expectedErrorMsg: ErrRemoteRepoReferenceContainsReservedTag.Error(),
		},
		{
			name:             "FullyQualified/WithEmptyTag/Invalid",
			reference:        "docker.io/library/archlinux:",
			expectedErrorMsg: ErrTagInvalidFormat.Error(),
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
			name:             "FullyQualified/InvalidTag",
			reference:        "docker.io/library/archlinux:...",
			expectedErrorMsg: "failed to parse repository tag: " + ErrTagInvalidFormat.Error(),
		},
		{
			name:             "InvalidDomain",
			reference:        ".docker.io/library/archlinux:latest",
			expectedErrorMsg: "failed to parse repository name: " + ErrReferenceInvalidFormat.Error(),
		},
		{
			name:             "InvalidName",
			reference:        "docker.io/library/§archlinux§:latest",
			expectedErrorMsg: "failed to parse repository name: " + ErrReferenceInvalidFormat.Error(),
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
			if test.expectedErrorMsg != "" {
				require.Error(t, err)
				require.Equal(t, err.Error(), test.expectedErrorMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedName, ref.Name())
			require.Equal(t, test.expectedReference, ref.AbsoluteReference())
		})
	}
}
