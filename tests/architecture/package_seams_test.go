package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const modulePath = "tangled.org/arabica.social/arabica"

func TestProtocolPackagesDoNotImportApplication(t *testing.T) {
	imports := protocolAppImports(t)
	assert.Empty(t, imports, "protocol and record-store contracts must not import application behavior")
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

func protocolAppImports(t *testing.T) map[string]struct{} {
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
			if rel == ".git" || rel == ".jj" || rel == "internal/arabica" {
				return filepath.SkipDir
			}
			return nil
		}

		rel := strings.TrimPrefix(path, "../../")
		if !(strings.HasPrefix(rel, "internal/atproto/") || strings.HasPrefix(rel, "internal/records/")) || !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
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
	for _, prefix := range []string{"arabica", "handlers", "routing", "server", "web"} {
		if strings.HasPrefix(importPath, modulePath+"/internal/"+prefix+"/") || importPath == modulePath+"/internal/"+prefix {
			return true
		}
	}
	return false
}
