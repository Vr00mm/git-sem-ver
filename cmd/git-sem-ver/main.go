// Command git-sem-ver generates a semantic version from the current GitFlow
// branch context. It reads GITHUB_REF_TYPE and GITHUB_REF_NAME to determine
// the branch kind, finds the latest semver tag reachable from HEAD, and
// outputs a version string to stdout and to $GITHUB_OUTPUT.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Vr00mm/git-sem-ver/internal/git"
	"github.com/Vr00mm/git-sem-ver/internal/gitctx"
	"github.com/Vr00mm/git-sem-ver/internal/semver"
)

// osExit is a variable so tests can replace it without killing the process.
var osExit = os.Exit

func main() {
	if err := execute(os.Stdout, os.Stderr); err != nil {
		osExit(1)
	}
}

func execute(out, errOut io.Writer) error {
	version, err := run()
	if err != nil {
		fmt.Fprintln(errOut, "error:", err) //nolint:errcheck // stderr write failure is unactionable
		return err
	}
	fmt.Fprintln(out, version) //nolint:errcheck // stdout write failure is unactionable
	if err := writeGithubOutput(version); err != nil {
		fmt.Fprintln(errOut, "warning: could not write to GITHUB_OUTPUT:", err) //nolint:errcheck // stderr write failure is unactionable
	}
	return nil
}

func run() (string, error) {
	ctx := gitctx.Load()

	// On a tag: the version IS the tag — return it directly with no pre-release.
	if ctx.Kind == gitctx.KindTag {
		v, err := semver.Parse(ctx.RefName)
		if err != nil {
			return "", fmt.Errorf("parsing tag %q as semver: %w", ctx.RefName, err)
		}
		return v.String(), nil
	}

	info, err := git.ReadInfo(".")
	if err != nil {
		return "", fmt.Errorf("reading git repository: %w", err)
	}

	// info.Base is the zero version (0.0.0) when no tag exists — safe to use directly.
	next := bump(info.Base, ctx.Bump)
	pre := preRelease(ctx, info.CommitCount)
	return next.WithPreRelease(pre).String(), nil
}

// bump applies the requested BumpKind to base.
func bump(base semver.Version, kind gitctx.BumpKind) semver.Version {
	switch kind {
	case gitctx.BumpMajor:
		return base.BumpMajor()
	case gitctx.BumpMinor:
		return base.BumpMinor()
	default:
		return base.BumpPatch()
	}
}

// preRelease builds the pre-release suffix for the given context and commit count.
func preRelease(ctx gitctx.Context, n int) string {
	switch ctx.Kind {
	case gitctx.KindMain:
		return fmt.Sprintf("rc.%d", n)
	case gitctx.KindDevelop:
		return fmt.Sprintf("dev.%d", n)
	case gitctx.KindFeature:
		return fmt.Sprintf("feat.%s.%d", ctx.ShortName, n)
	case gitctx.KindFix:
		return fmt.Sprintf("fix.%s.%d", ctx.ShortName, n)
	case gitctx.KindHotfix:
		return fmt.Sprintf("hotfix.%s.%d", ctx.ShortName, n)
	case gitctx.KindRelease:
		return fmt.Sprintf("beta.%d", n)
	default:
		if ctx.ShortName != "" {
			return fmt.Sprintf("branch.%s.%d", ctx.ShortName, n)
		}
		return fmt.Sprintf("branch.%d", n)
	}
}

// writeGithubOutput appends version=<value> to the file pointed to by
// GITHUB_OUTPUT, which is how GitHub Actions step outputs work.
func writeGithubOutput(version string) error {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return nil
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening GITHUB_OUTPUT: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing GITHUB_OUTPUT: %w", cerr)
		}
	}()

	_, err = fmt.Fprintf(f, "version=%s\n", version)
	if err != nil {
		return fmt.Errorf("writing GITHUB_OUTPUT: %w", err)
	}
	return nil
}
