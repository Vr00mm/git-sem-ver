// Package gitctx parses GitHub Actions environment variables to determine
// the current GitFlow branch context and version bump strategy.
package gitctx

import (
	"os"
	"strings"
)

// BranchKind represents the GitFlow branch type.
type BranchKind int

// Branch kind constants — ordered by GitFlow branch type.
const (
	KindMain    BranchKind = iota // main or master → rc pre-release.
	KindDevelop                   // develop         → dev pre-release.
	KindFeature                   // feat/*          → minor bump, feat pre-release.
	KindFix                       // fix/* bugfix/*  → patch bump, fix pre-release.
	KindHotfix                    // hotfix/*        → patch bump, hotfix pre-release.
	KindRelease                   // release/*       → beta pre-release.
	KindTag                       // tag             → clean version.
	KindOther                     // anything else   → branch pre-release.
)

// BumpKind represents the semver component to increment.
type BumpKind int

// Bump kind constants — ordered patch → minor → major.
const (
	BumpPatch BumpKind = iota // default: increment patch (e.g. 1.2.3 → 1.2.4).
	BumpMinor                 // increment minor, reset patch (e.g. 1.2.3 → 1.3.0).
	BumpMajor                 // increment major, reset minor and patch (e.g. 1.2.3 → 2.0.0).
)

// Context holds the parsed GitHub Actions context.
type Context struct {
	RefType   string     // "branch" or "tag".
	RefName   string     // e.g. "main", "feat/my-feature", "v1.2.3".
	SHA       string     // GITHUB_SHA.
	Kind      BranchKind // resolved branch kind.
	ShortName string     // slugified suffix for feat/fix/hotfix/release/other.
	Bump      BumpKind   // resolved bump type (may be overridden by GIT_SEM_VER_BUMP).
}

const maxShortNameLen = 20

// Load reads GITHUB_* env vars and returns the current context.
// GIT_SEM_VER_BUMP can be set to "major", "minor", or "patch" to override
// the default bump strategy derived from the branch name.
func Load() Context {
	ctx := Context{
		RefType: os.Getenv("GITHUB_REF_TYPE"),
		RefName: os.Getenv("GITHUB_REF_NAME"),
		SHA:     os.Getenv("GITHUB_SHA"),
	}

	ctx.Kind = resolveKind(ctx.RefType, ctx.RefName)
	ctx.ShortName = resolveShortName(ctx.Kind, ctx.RefName)
	ctx.Bump = resolveBump(ctx.Kind)

	if override := os.Getenv("GIT_SEM_VER_BUMP"); override != "" {
		ctx.Bump = parseBumpOverride(override, ctx.Bump)
	}

	return ctx
}

func resolveKind(refType, refName string) BranchKind {
	if refType == "tag" {
		return KindTag
	}
	switch {
	case refName == "main" || refName == "master":
		return KindMain
	case refName == "develop" || refName == "development":
		return KindDevelop
	case strings.HasPrefix(refName, "feat/") || strings.HasPrefix(refName, "feature/"):
		return KindFeature
	case strings.HasPrefix(refName, "fix/") || strings.HasPrefix(refName, "bugfix/"):
		return KindFix
	case strings.HasPrefix(refName, "hotfix/"):
		return KindHotfix
	case strings.HasPrefix(refName, "release/"):
		return KindRelease
	default:
		return KindOther
	}
}

func resolveShortName(kind BranchKind, refName string) string {
	switch kind {
	case KindFeature:
		return slugify(stripPrefix(refName, "feat/", "feature/"), maxShortNameLen)
	case KindFix:
		return slugify(stripPrefix(refName, "fix/", "bugfix/"), maxShortNameLen)
	case KindHotfix:
		return slugify(stripPrefix(refName, "hotfix/"), maxShortNameLen)
	case KindRelease:
		return slugify(stripPrefix(refName, "release/"), maxShortNameLen)
	case KindOther:
		return slugify(refName, maxShortNameLen)
	default:
		return ""
	}
}

func resolveBump(kind BranchKind) BumpKind {
	switch kind {
	case KindFeature:
		return BumpMinor
	default:
		return BumpPatch
	}
}

func parseBumpOverride(s string, fallback BumpKind) BumpKind {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "major":
		return BumpMajor
	case "minor":
		return BumpMinor
	case "patch":
		return BumpPatch
	default:
		return fallback
	}
}

// stripPrefix removes the first matching prefix from s.
func stripPrefix(s string, prefixes ...string) string {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return s[len(p):]
		}
	}
	return s
}

// slugify lowercases s, replaces non-alphanumeric characters with hyphens,
// collapses consecutive hyphens, strips leading/trailing hyphens,
// and truncates to maxLen.
func slugify(s string, maxLen int) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevHyphen := true // treat start as hyphen to avoid leading hyphen.
	for _, r := range s {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteRune('-')
			prevHyphen = true
		}
	}
	result := strings.TrimRight(b.String(), "-")
	if len(result) <= maxLen {
		return result
	}
	return strings.TrimRight(result[:maxLen], "-")
}
