package version

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Paths are relative to this package dir (go test sets cwd to the package).
var (
	makefilePath        = filepath.Join("..", "..", "Makefile")
	goModPath           = filepath.Join("..", "..", "go.mod")
	releaseWorkflowPath = filepath.Join("..", "..", ".github", "workflows", "release.yml")
)

// makeVar returns the value of a Makefile variable, joining backslash
// continuations. Enough for the `NAME := ...` / `NAME = ...` lines guarded here.
func makeVar(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("read %s: %v", makefilePath, err)
	}
	lines := strings.Split(string(data), "\n")
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(name) + `\s*:?=\s*(.*)$`)
	var out []string
	on := false
	for _, line := range lines {
		if !on {
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			on = true
			out = append(out, m[1])
		} else {
			out = append(out, line)
		}
		if on && !strings.HasSuffix(line, "\\") {
			break
		}
	}
	if !on {
		t.Fatalf("%s declares no %s", makefilePath, name)
	}
	return strings.Join(out, " ")
}

// TestDistStampsAllVersionVars is the drift guard for release build metadata:
// every exported var in this package must be injected by the Makefile's `dist`
// ldflags, under this package's real import path.
//
// Without it the failure is invisible until after a release: the binary builds
// and runs, `kubecom version` just reports the placeholder ("commit none, built
// unknown"), and a published channel build cannot be rebuilt in place (D173
// pt 1). Adding a var here without wiring it there now fails `make check`. The
// same guard was goreleaser-shaped until goreleaser was replaced by `make dist`
// + engram channels; the property, not the tool, is what is checked.
func TestDistStampsAllVersionVars(t *testing.T) {
	pkgPath := modulePath(t) + "/internal/version"
	flags := strings.ReplaceAll(makeVar(t, "DIST_LDFLAGS"), "$(VERSION_PKG)", pkgPath)

	vars := exportedVars(t)
	if len(vars) == 0 {
		t.Fatal("no exported vars found in version.go — the guard is not reading the package")
	}
	for _, name := range vars {
		prefix := "-X " + pkgPath + "." + name + "="
		if !strings.Contains(flags, prefix) {
			t.Errorf("DIST_LDFLAGS does not set %s (want %q<value>)", name, prefix)
		}
	}

	// The launcher rides the seed's version (D295): it is stamped with the same
	// metadata as the binary by reusing DIST_LDFLAGS, so a release's launcher and
	// seed name one version.
	launcher := makeVar(t, "LAUNCHER_FLAGS")
	if !strings.Contains(launcher, "$(DIST_LDFLAGS)") {
		t.Errorf("LAUNCHER_FLAGS does not reuse DIST_LDFLAGS; the launcher must ride the seed's version (D295): %q", launcher)
	}
}

// TestReleaseWorkflowPublishesTheChannel guards the properties of the release
// workflow that cannot be caught by running it, because the run that would
// catch them is the tag push — cached permanently by the module proxy and
// impossible to retry (D173 pt 1).
//
//  1. Pushes to v1 publish the dev channel; a day-named tag publishes stable —
//     the margin/engram pipeline kubecom adopted in place of goreleaser.
//  2. The publish signs in to AWS with OIDC (no stored secret) and reads the
//     build back through the CDN the way `kubecom update` will.
//  3. goreleaser is gone: leaving it would be a second, silently-drifting
//     release path.
func TestReleaseWorkflowPublishesTheChannel(t *testing.T) {
	data, err := os.ReadFile(releaseWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", releaseWorkflowPath, err)
	}
	wf := string(data)

	if strings.Contains(wf, "goreleaser") {
		t.Errorf("%s still names goreleaser; the release path is `make dist` + engram", releaseWorkflowPath)
	}
	for _, want := range []string{
		"branches: [main]",
		"[0-9][0-9].[0-9][0-9].[0-9][0-9]",
		"make dist",
		"engram publish",
		"--project kubecom",
		"engram verify",
		"aws-actions/configure-aws-credentials",
		"uses: ./.github/workflows/ci.yml",
	} {
		if !strings.Contains(wf, want) {
			t.Errorf("%s is missing %q", releaseWorkflowPath, want)
		}
	}
}

// modulePath reads the module line from go.mod, so a module rename fails here
// rather than silently producing ldflags that match no package.
func modulePath(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read %s: %v", goModPath, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	t.Fatalf("%s has no module line", goModPath)
	return ""
}

// exportedVars returns the exported package-level var names declared in
// version.go — the set that must be injectable at release time.
func exportedVars(t *testing.T) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "version.go", nil, 0)
	if err != nil {
		t.Fatalf("parse version.go: %v", err)
	}
	var names []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, ident := range value.Names {
				if ident.IsExported() {
					names = append(names, ident.Name)
				}
			}
		}
	}
	return names
}
