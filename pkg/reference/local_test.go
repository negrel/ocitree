package reference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalReference(t *testing.T) {
	for _, test := range []struct {
		name              string
		reference         string
		expectedName      string
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
			expectedReference: "negrel.dev/archlinux:latest",
		},
		{
			name:              "FullyQualified/WithCustomDomain/Valid",
			reference:         "negrel.dev/library/archlinux:latest",
			expectedName:      "negrel.dev/library/archlinux",
			expectedReference: "negrel.dev/library/archlinux:latest",
		},
		{
			name:              "FullyQualified/WithCustomTag/Valid",
			reference:         "docker.io/library/archlinux:edge",
			expectedName:      "docker.io/library/archlinux",
			expectedReference: "docker.io/library/archlinux:edge",
		},
		{
			name:              "FullyQualified/WithHEADTag/Valid",
			reference:         "docker.io/library/archlinux:HEAD",
			expectedName:      "docker.io/library/archlinux",
			expectedReference: "docker.io/library/archlinux:HEAD",
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
			expectedReference: "localhost/archlinux:edge",
		},
		{
			name:              "FullyQualified/Valid",
			reference:         "docker.io/library/archlinux:edge",
			expectedName:      "docker.io/library/archlinux",
			expectedReference: "docker.io/library/archlinux:edge",
		},
		{
			name:             "FullyQualified/InvalidTag",
			reference:        "docker.io/library/archlinux:...",
			expectedErrorMsg: ErrTagInvalidFormat.Error(),
		},
		{
			name:             "InvalidDomain",
			reference:        ".docker.io/library/archlinux:latest",
			expectedErrorMsg: ErrReferenceInvalidFormat.Error(),
		},
		{
			name:             "InvalidName",
			reference:        "docker.io/library/§archlinux§:latest",
			expectedErrorMsg: ErrReferenceInvalidFormat.Error(),
		},
		{
			name:              "MissingDomain/Valid",
			reference:         "archlinux:latest",
			expectedName:      "docker.io/library/archlinux",
			expectedReference: "docker.io/library/archlinux:latest",
		},
		{
			name:              "MissingDomainAndTag/Valid",
			reference:         "archlinux",
			expectedName:      "docker.io/library/archlinux",
			expectedReference: "docker.io/library/archlinux:HEAD",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ref, err := LocalFromString(test.reference)
			if test.expectedErrorMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrorMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expectedName, ref.Name())
			require.Equal(t, test.expectedReference, ref.AbsoluteReference())
		})
	}
}
