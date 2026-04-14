package gitctx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"my-feature", 20, "my-feature"},
		{"My Feature", 20, "my-feature"},
		{"TICKET-123_fix_something", 20, "ticket-123-fix-somet"},
		{"---leading", 20, "leading"},
		{"trailing---", 20, "trailing"},
		{"a--b", 20, "a-b"},
		{"hello world", 5, "hello"},
		{"hello-", 6, "hello"},
		{"", 20, ""},
		{"123", 20, "123"},
		// truncation must not leave a trailing hyphen
		{"abcde-fghij-klmno-pqrst", 10, "abcde-fghi"},
		{"abcde-", 6, "abcde"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, slugify(tt.input, tt.maxLen))
		})
	}
}

func TestResolveKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		refType string
		refName string
		want    BranchKind
	}{
		{"tag", "v1.2.3", KindTag},
		{"branch", "main", KindMain},
		{"branch", "master", KindMain},
		{"branch", "develop", KindDevelop},
		{"branch", "development", KindDevelop},
		{"branch", "feat/new-api", KindFeature},
		{"branch", "feature/new-api", KindFeature},
		{"branch", "fix/login-bug", KindFix},
		{"branch", "bugfix/login-bug", KindFix},
		{"branch", "hotfix/critical", KindHotfix},
		{"branch", "release/1.2.0", KindRelease},
		{"branch", "chore/deps", KindOther},
		{"branch", "dependabot/npm/lodash", KindOther},
	}

	for _, tt := range tests {
		t.Run(tt.refName, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveKind(tt.refType, tt.refName))
		})
	}
}

func TestResolveShortName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind    BranchKind
		refName string
		want    string
	}{
		{KindFeature, "feat/my-feature", "my-feature"},
		{KindFeature, "feature/TICKET-123_new_payment_flow", "ticket-123-new-payme"},
		{KindFix, "fix/login-redirect", "login-redirect"},
		{KindFix, "bugfix/null-pointer", "null-pointer"},
		{KindHotfix, "hotfix/prod-crash", "prod-crash"},
		{KindRelease, "release/2.0.0", "2-0-0"},
		{KindOther, "chore/update-deps", "chore-update-deps"},
		// kinds without a short name return empty string
		{KindMain, "main", ""},
		{KindDevelop, "develop", ""},
		{KindTag, "v1.2.3", ""},
	}

	for _, tt := range tests {
		t.Run(tt.refName, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveShortName(tt.kind, tt.refName))
		})
	}
}

func TestStripPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		prefixes []string
		want     string
	}{
		{"feat/my-feature", []string{"feat/", "feature/"}, "my-feature"},
		{"feature/my-feature", []string{"feat/", "feature/"}, "my-feature"},
		// no prefix matches → input returned unchanged
		{"chore/cleanup", []string{"feat/", "fix/"}, "chore/cleanup"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stripPrefix(tt.input, tt.prefixes...))
		})
	}
}

func TestParseBumpOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		fallback BumpKind
		want     BumpKind
	}{
		{"major", BumpPatch, BumpMajor},
		{"MAJOR", BumpPatch, BumpMajor},
		{"minor", BumpPatch, BumpMinor},
		{"patch", BumpMinor, BumpPatch},
		{"", BumpMinor, BumpMinor},
		{"invalid", BumpPatch, BumpPatch},
		{"  major  ", BumpPatch, BumpMajor}, // trims whitespace
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseBumpOverride(tt.input, tt.fallback))
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name         string
		refType      string
		refName      string
		sha          string
		bumpOverride string
		wantKind     BranchKind
		wantBump     BumpKind
		wantShort    string
	}{
		{
			name:      "main branch",
			refType:   "branch",
			refName:   "main",
			wantKind:  KindMain,
			wantBump:  BumpPatch,
			wantShort: "",
		},
		{
			name:      "feature branch gets minor bump",
			refType:   "branch",
			refName:   "feat/new-login",
			wantKind:  KindFeature,
			wantBump:  BumpMinor,
			wantShort: "new-login",
		},
		{
			name:      "fix branch gets patch bump",
			refType:   "branch",
			refName:   "fix/crash",
			wantKind:  KindFix,
			wantBump:  BumpPatch,
			wantShort: "crash",
		},
		{
			name:         "bump override to major",
			refType:      "branch",
			refName:      "feat/breaking",
			bumpOverride: "major",
			wantKind:     KindFeature,
			wantBump:     BumpMajor,
			wantShort:    "breaking",
		},
		{
			name:      "tag ref",
			refType:   "tag",
			refName:   "v1.2.3",
			wantKind:  KindTag,
			wantBump:  BumpPatch,
			wantShort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REF_TYPE", tt.refType)
			t.Setenv("GITHUB_REF_NAME", tt.refName)
			t.Setenv("GITHUB_SHA", tt.sha)
			t.Setenv("GIT_SEM_VER_BUMP", tt.bumpOverride)

			ctx := Load()

			assert.Equal(t, tt.wantKind, ctx.Kind)
			assert.Equal(t, tt.wantBump, ctx.Bump)
			assert.Equal(t, tt.wantShort, ctx.ShortName)
			assert.Equal(t, tt.refName, ctx.RefName)
		})
	}
}
