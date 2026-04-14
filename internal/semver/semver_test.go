package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"1.2.3", Version{1, 2, 3, ""}, false},
		{"v1.2.3", Version{1, 2, 3, ""}, false},
		{"0.0.0", Version{0, 0, 0, ""}, false},
		{"10.20.30", Version{10, 20, 30, ""}, false},
		// pre-release and build metadata are stripped during parse
		{"1.2.3-rc.1", Version{1, 2, 3, ""}, false},
		{"1.2.3+build.42", Version{1, 2, 3, ""}, false},
		// errors
		{"1.2", Version{}, true},
		{"1.2.3.4", Version{}, true},
		{"x.2.3", Version{}, true},  // invalid major
		{"v1.x.3", Version{}, true}, // invalid minor
		{"1.2.x", Version{}, true},  // invalid patch
		{"", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBump(t *testing.T) {
	t.Parallel()

	base := Version{1, 2, 3, ""}

	assert.Equal(t, Version{2, 0, 0, ""}, base.BumpMajor(), "major resets minor and patch")
	assert.Equal(t, Version{1, 3, 0, ""}, base.BumpMinor(), "minor resets patch")
	assert.Equal(t, Version{1, 2, 4, ""}, base.BumpPatch())
}

func TestBumpFromZero(t *testing.T) {
	t.Parallel()

	zero := Version{}
	assert.Equal(t, Version{1, 0, 0, ""}, zero.BumpMajor())
	assert.Equal(t, Version{0, 1, 0, ""}, zero.BumpMinor())
	assert.Equal(t, Version{0, 0, 1, ""}, zero.BumpPatch())
}

func TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v    Version
		want string
	}{
		{Version{1, 2, 3, ""}, "1.2.3"},
		{Version{1, 2, 3, "rc.1"}, "1.2.3-rc.1"},
		{Version{0, 0, 0, "dev.5"}, "0.0.0-dev.5"},
		{Version{2, 0, 0, "feat.new-api.12"}, "2.0.0-feat.new-api.12"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.v.String())
		})
	}
}

func TestWithPreRelease(t *testing.T) {
	t.Parallel()

	v := Version{1, 2, 3, ""}
	got := v.WithPreRelease("rc.1")

	assert.Equal(t, Version{1, 2, 3, "rc.1"}, got)
	assert.Empty(t, v.PreRelease, "original must not be mutated")
}
