package libocitree

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseReference(t *testing.T) {
	for _, test := range []struct {
		name           string
		reference      string
		expectedError  string
		expectedName   string
		expectedTag    string
		expectedOffset uint
		expectPanic    bool
	}{
		{
			name:           "EmptyReference/Invalid",
			reference:      "",
			expectedError: "failed to parse repository reference: invalid reference format",
		},
		{
			name:           "LocalAbsoluteReference/Valid",
			reference:      "localhost/archlinux:latest",
			expectedName:   "localhost/archlinux",
			expectedTag:    "latest",
			expectedOffset: 0,
		},
		{
			name:           "AbsoluteReference/Valid",
			reference:      "docker.io/library/archlinux:latest",
			expectedName:   "docker.io/library/archlinux",
			expectedTag:    "latest",
			expectedOffset: 0,
		},
		{
			name:          "AbsoluteReference/InvalidTag",
			reference:     "docker.io/library/archlinux:...",
			expectedError: "failed to parse repository reference: invalid reference format",
		},
		{
			name:          "AbsoluteReference/InvalidDomain",
			reference:     ".docker.io/library/archlinux:latest",
			expectedError: "failed to parse repository reference: invalid reference format",
		},
		{
			name:          "AbsoluteReference/InvalidName",
			reference:     "docker.io/library/§archlinux§:latest",
			expectedError: "failed to parse repository reference: invalid reference format",
		},
		{
			name:           "AbsoluteReferenceWithMissingTag/Valid",
			reference:      "docker.io/library/archlinux",
			expectedName:   "docker.io/library/archlinux",
			expectedTag:    "HEAD",
			expectedOffset: 0,
		},
		{
			name:           "AbsoluteReferenceWithMissingDomain/Valid",
			reference:      "archlinux:latest",
			expectedName:   "docker.io/library/archlinux",
			expectedTag:    "latest",
			expectedOffset: 0,
		},
		{
			name:           "AbsoluteReferenceWithMissingDomainAndTag/Valid",
			reference:      "archlinux",
			expectedName:   "docker.io/library/archlinux",
			expectedTag:    "HEAD",
			expectedOffset: 0,
		},
		{
			name:           "RelativeReferenceWithTilde/Valid",
			reference:      "docker.io/library/archlinux:latest~3",
			expectedName:   "docker.io/library/archlinux",
			expectedTag:    "latest",
			expectedOffset: 3,
		},
		{
			name:           "RelativeReferenceWithCircumflex/Valid",
			reference:      "docker.io/library/archlinux:latest^^",
			expectedName:   "docker.io/library/archlinux",
			expectedTag:    "latest",
			expectedOffset: 2,
		},
		{
			name:          "RelativeReferenceWithCircumflexButWithoutTag/Invalid",
			reference:     "docker.io/library/archlinux:^^",
			expectedError: "failed to parse repository reference: invalid reference format",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ref, err := ParseReference(test.reference)
			if len(test.expectedError) != 0 {
				require.Error(t, err)
				require.Equal(t, test.expectedError, err.Error())
				return
			}
			require.NoError(t, err, "no parse reference expected")

			require.Equal(t, test.expectedName, ref.Name(), "reference name doesn't match expected value")
			require.Equal(t, test.expectedTag, ref.Tag(), "reference tag doesn't match expected value")
			require.Equal(t, test.expectedOffset, ref.Offset(), "reference offset doesn't match expected value")
		})
	}
}

func TestReferenceString(t *testing.T) {
	for _, test := range []struct {
		name           string
		reference      string
		expectedString string
	}{
		{
			name:           "FullyQualified",
			reference:      "docker.io/library/archlinx:latest",
			expectedString: "docker.io/library/archlinx:latest",
		},
		{
			name:           "Minimal",
			reference:      "archlinux",
			expectedString: "docker.io/library/archlinux:HEAD",
		},
		{
			name:           "RelativeWithTilde",
			reference:      "archlinux:HEAD~5",
			expectedString: "docker.io/library/archlinux:HEAD~5",
		},
		{
			name:           "RelativeWithCircumflex",
			reference:      "archlinux:latest^^^^^",
			expectedString: "docker.io/library/archlinux:latest~5",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ref, err := ParseReference(test.reference)
			require.NoError(t, err)

			require.Equal(t, test.expectedString, ref.String())
		})
	}
}

func TestReferenceToRemoteReference(t *testing.T) {
	for _, test := range []struct {
		name                 string
		reference            string
		expectedRemoteString string
	}{
		{
			name:                 "ValidRemoteReference",
			reference:            "docker.io/library/archlinux:latest",
			expectedRemoteString: "docker.io/library/archlinux:latest",
		},
		{
			name:                 "ValidRemoteReferenceWithEdgeTag",
			reference:            "docker.io/library/archlinux:edge",
			expectedRemoteString: "docker.io/library/archlinux:edge",
		},
		{
			name:                 "WithHEAD",
			reference:            "docker.io/library/archlinux:HEAD",
			expectedRemoteString: "docker.io/library/archlinux:latest",
		},
		{
			name:                 "Relative",
			reference:            "docker.io/library/archlinux:edge^^",
			expectedRemoteString: "docker.io/library/archlinux:edge",
		},
		{
			name:                 "WithRelativeHEAD",
			reference:            "docker.io/library/archlinux:HEAD^^",
			expectedRemoteString: "docker.io/library/archlinux:latest",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ref, err := ParseReference(test.reference)
			require.NoError(t, err)

			remoteRef := ref.ToRemoteReference()
			require.Equal(t, test.expectedRemoteString, remoteRef.String())
		})
	}
}
