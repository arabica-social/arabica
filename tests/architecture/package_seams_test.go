package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
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
	"internal/firehose/index.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/firehose/index.go imports tangled.org/arabica.social/arabica/internal/oolong/entities",
	"internal/firehose/notifications.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/feed.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/handlers.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/notifications.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/handlers/pages.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/ogcard/brew.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
	"internal/ogcard/entities.go imports tangled.org/arabica.social/arabica/internal/arabica/entities",
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

func TestDomainEntityRegistryDoesNotImportTempl(t *testing.T) {
	imports := importsForDir(t, "../../internal/entities")
	_, importsTempl := imports["github.com/a-h/templ"]
	assert.False(t, importsTempl, "internal/entities is domain metadata; feed rendering belongs in app-owned web packages")
}

func TestDomainEntityDescriptorDoesNotOwnFeedActions(t *testing.T) {
	fields := structFields(t, "../../internal/entities/entities.go", "Descriptor")
	assert.NotContains(t, fields, "Noun")
	assert.NotContains(t, fields, "URLPath")
	assert.NotContains(t, fields, "GetField")
	assert.NotContains(t, fields, "RecordToModel")
	assert.NotContains(t, fields, "RKey")
	assert.NotContains(t, fields, "DisplayTitle")
	assert.NotContains(t, fields, "ResolveRefs")
	assert.NotContains(t, fields, "RenderFeedContent")
	assert.NotContains(t, fields, "FeedCardCompact")
	assert.NotContains(t, fields, "FeedFilterLabel")
	assert.NotContains(t, fields, "EditURL")
	assert.NotContains(t, fields, "EditModalURL")
}

func TestFeedPageDoesNotReadDescriptorRouteFields(t *testing.T) {
	content, err := os.ReadFile("../../internal/web/pages/feed.templ")
	assert.NoError(t, err)

	source := string(content)
	assert.NotContains(t, source, "entities.Get(")
	assert.NotContains(t, source, ".Noun")
	assert.NotContains(t, source, ".URLPath")
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

func importsForDir(t *testing.T, dir string) map[string]struct{} {
	t.Helper()

	imports := map[string]struct{}{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		assert.NoError(t, err)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		assert.NoError(t, parseErr)
		if parseErr != nil {
			return parseErr
		}
		for _, spec := range file.Imports {
			imports[unquoteImportPath(t, spec)] = struct{}{}
		}
		return nil
	})
	assert.NoError(t, err)
	return imports
}

func structFields(t *testing.T, path, typeName string) map[string]struct{} {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	assert.NoError(t, err)
	if err != nil {
		return nil
	}

	fields := map[string]struct{}{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			assert.True(t, ok, "%s should be a struct", typeName)
			if !ok {
				return fields
			}
			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					fields[name.Name] = struct{}{}
				}
			}
			return fields
		}
	}

	assert.Fail(t, "struct not found", typeName)
	return fields
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
