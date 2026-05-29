package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const modulePath = "tangled.org/arabica.social/arabica"

var existingSharedAppImports = []string{
	"internal/atproto/store.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/atproto/store_arabica_codecs.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/atproto/store_generic.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/database/store.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/database/store_mock.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/feed/service.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/firehose/index.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/firehose/index.go imports tangled.org/arabica.social/arabica/internal/oolong/entities",
	"internal/firehose/notifications.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/admin.go imports tangled.org/arabica.social/arabica/internal/arabica/web/pages",
	"internal/handlers/feed.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/feed.go imports tangled.org/arabica.social/arabica/internal/oolong/entities",
	"internal/handlers/handlers.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/notifications.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/pages.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/pages.go imports tangled.org/arabica.social/arabica/internal/oolong/web/pages",
	"internal/handlers/profile.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/profile.go imports tangled.org/arabica.social/arabica/internal/arabica/web/components",
	"internal/handlers/profile.go imports tangled.org/arabica.social/arabica/internal/arabica/web/pages",
	"internal/ogcard/brew.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/ogcard/entities.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/onboarding/readiness.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/routing/routing.go imports tangled.org/arabica.social/arabica/internal/arabica/handlers",
	"internal/routing/routing.go imports tangled.org/arabica.social/arabica/internal/oolong/handlers",
	"internal/web/bff/helpers.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/web/components/incomplete_records_templ.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/web/components/shared_templ.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/web/pages/notifications_templ.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/web/pages/settings_templ.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
}

func TestSharedPackagesDoNotAddAppImports(t *testing.T) {
	actual := sharedAppImports(t)
	expected := stringSet(existingSharedAppImports)

	unexpected := difference(actual, expected)
	stale := difference(expected, actual)

	assert.Empty(t, unexpected, "new app-specific imports from shared packages are forbidden; move behavior to internal/arabica or internal/oolong, or deliberately update the baseline while paying down this seam")
	assert.Empty(t, stale, "remove fixed imports from existingSharedAppImports so the seam guard keeps ratcheting down")
}

func sharedAppImports(t *testing.T) map[string]struct{} {
	t.Helper()

	imports := map[string]struct{}{}
	fset := token.NewFileSet()
	err := filepath.WalkDir("../..", func(path string, d fs.DirEntry, err error) error {
		assert.NoError(t, err)
		if err != nil {
			return err
		}

		path = filepath.ToSlash(path)
		if d.IsDir() {
			if path == "../.." {
				return nil
			}
			rel := strings.TrimPrefix(path, "../../")
			if rel == ".git" || rel == ".jj" || rel == "internal/arabica" || rel == "internal/oolong" {
				return filepath.SkipDir
			}
			return nil
		}

		rel := strings.TrimPrefix(path, "../../")
		if !strings.HasPrefix(rel, "internal/") || !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		assert.NoError(t, parseErr)
		if parseErr != nil {
			return parseErr
		}

		for _, spec := range file.Imports {
			importPath := unquoteImportPath(t, spec)
			if isAppImport(importPath) {
				imports[rel+" imports "+importPath] = struct{}{}
			}
		}
		return nil
	})
	assert.NoError(t, err)
	return imports
}

func unquoteImportPath(t *testing.T, spec *ast.ImportSpec) string {
	t.Helper()
	importPath, err := strconv.Unquote(spec.Path.Value)
	assert.NoError(t, err)
	return importPath
}

func isAppImport(importPath string) bool {
	return strings.HasPrefix(importPath, modulePath+"/internal/arabica") ||
		strings.HasPrefix(importPath, modulePath+"/internal/oolong")
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func difference(left, right map[string]struct{}) []string {
	var diff []string
	for value := range left {
		if _, ok := right[value]; !ok {
			diff = append(diff, value)
		}
	}
	sort.Strings(diff)
	return diff
}
