package main

import (
	"errors"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

type testFile string

func (t testFile) Name() string       { return string(t) }
func (t testFile) Size() int64        { return -1 }
func (t testFile) Mode() fs.FileMode  { return fs.ModeType }
func (t testFile) ModTime() time.Time { return time.Now() }
func (t testFile) IsDir() bool        { return false }
func (t testFile) Sys() any           { return t }

type testFS map[string]string

func (t testFS) OpenFile(name string) (io.ReadCloser, error) {
	contents, ok := t[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return io.NopCloser(strings.NewReader(contents)), nil
}

func (t testFS) IsDir(name string) bool {
	return name == "."
}

func (t testFS) ReadDir(dir string) ([]fs.FileInfo, error) {
	entries := make([]fs.FileInfo, 0, len(t))

	for name := range t {
		entries = append(entries, testFile(name))
	}

	return entries, nil
}

func (t testFS) ReadFile(name string) ([]byte, error) {
	contents, ok := t[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return []byte(contents), nil
}

func TestListFiles(t *testing.T) {
	badOS := "windows"

	if runtime.GOOS == "windows" {
		badOS = "darwin"
	}

	tfs := testFS{
		"a.go":                      "package main\n\nconst a = 1",
		"a_" + badOS + ".go":        "package main\n\nconst b = 1",
		"a_" + runtime.GOOS + ".go": "package main\n\nconst b = 2",
		"go.mod":                    "module example.com/main\n\ngo 1.25.5",
		"a_test.go":                 "package main\n\nconst c = 3",
	}

	files, err := listGoFiles(&tfs)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	slices.Sort(files)

	expectedFiles := []string{"a.go", "a_" + runtime.GOOS + ".go"}

	if !slices.Equal(files, expectedFiles) {
		t.Errorf("expecting files %v, got %v", expectedFiles, files)
	}
}

func TestParsePackage(t *testing.T) {
	tfs := testFS{
		"a.go": "package main\n\ntype A struct {B int}",
	}

	m := moduleDetails{fset: token.NewFileSet()}

	if pkg, err := m.ParsePackage(tfs, "example.com/pkg"); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if a := pkg.Scope().Lookup("A"); a == nil {
		t.Error("expected type def, got nil")
	} else if pkgPath := a.Pkg().Path(); pkgPath != "example.com/pkg" {
		t.Errorf("expecting package path %q, got %q", "example.com/pkg", pkgPath)
	} else if as, ok := a.Type().Underlying().(*types.Struct); !ok {
		t.Error("expected struct type")
	} else if nf := as.NumFields(); nf != 1 {
		t.Errorf("expected 1 field, got %d", nf)
	} else if name := as.Field(0).Name(); name != "B" {
		t.Errorf("expected field name %q, got %q", "B", name)
	} else if b, ok := as.Field(0).Type().Underlying().(*types.Basic); !ok {
		t.Error("expected basic type")
	} else if b.Kind() != types.Int {
		t.Errorf("expected type %d, got %v", types.Int, b.Kind())
	}

	if pkg, err := m.ParsePackage(tfs, "", "a.go"); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if a := pkg.Scope().Lookup("A"); a != nil {
		t.Errorf("expected no object, got %v", a)
	}
}

func TestParseFiles(t *testing.T) {
	tfs := testFS{
		"a.go": "package main\n\ntype A struct {B int}",
		"b.go": "package pkg\n\ntype D struct {E int}",
		"c.go": "package pkg\n\ntype F struct {G int}",
	}

	m := moduleDetails{fset: token.NewFileSet()}

	if files, err := m.parseFiles(".", tfs, []string{"a.go", "b.go", "c.go"}); !errors.Is(err, errMultiplePackages) {
		t.Errorf("expecting error errMultiplePackages, got %v", err)
	} else if files != nil {
		t.Errorf("expecting nil files, got %v", files)
	} else if files, err = m.parseFiles(".", tfs, []string{"b.go", "c.go"}); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if len(files) != 2 {
		t.Errorf("expecting 2 files, got %d", len(files))
	}
}

func TestParseModFile(t *testing.T) {
	tfs := testFS{
		"go.mod": `module vimagination.zapto.org/marshal

go 1.25.5

require (
	golang.org/x/mod v0.31.0
	golang.org/x/tools v0.40.0
)

require golang.org/x/sync v0.19.0 // indirect

replace golang.org/x/tools => somewhere.org/tools v0.1.0
`,
	}

	if pkg, err := parseModFile(tfs, ""); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if pkg.Module != "vimagination.zapto.org/marshal" {
		t.Errorf("expecting path %q, got %q", "vimagination.zapto.org/marshal", pkg.Module)
	} else if len(pkg.Imports) != 3 {
		t.Errorf("expecting 3 imports, got %d", len(pkg.Imports))
	} else if m := pkg.Imports["golang.org/x/mod"]; m.Path != "golang.org/x/mod" {
		t.Errorf("expecting url for %q to be %q, got %q", "golang.org/x/mod", "golang.org/x/mod", m.Path)
	} else if m.Version != "v0.31.0" {
		t.Errorf("expecting version for %q to be %q, got %q", "golang.org/x/mod", "v0.31.0", m.Version)
	} else if m = pkg.Imports["golang.org/x/tools"]; m.Path != "somewhere.org/tools" {
		t.Errorf("expecting url for %q to be %q, got %q", "golang.org/x/tools", "somewhere.org/tools", m.Path)
	} else if m.Version != "v0.1.0" {
		t.Errorf("expecting version for %q to be %q, got %q", "golang.org/x/tools", "v0.1.0", m.Version)
	}
}

func TestModCacheURL(t *testing.T) {
	im := importDetails{Base: "golang.org/x/sync", Version: "v0.19.0"}
	url, err := im.ModCacheURL()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if url != "https://proxy.golang.org/golang.org/x/sync/@v/v0.19.0.zip" {
		t.Errorf("expecting URL %q, got %q", "https://proxy.golang.org/golang.org/x/sync/@v/v0.19.0.zip", url)
	}
}

func TestImportResolve(t *testing.T) {
	tfs := testFS{
		"go.mod": `module vimagination.zapto.org/marshal

go 1.25.5

require (
	golang.org/x/mod v0.31.0
	golang.org/x/tools v0.40.0
	vimagination.zapto.org/httpreaderat v1.0.0
)

require golang.org/x/sync v0.19.0 // indirect

replace golang.org/x/tools => somewhere.org/tools v0.1.0

replace vimagination.zapto.org/httpreaderat v1.0.0 => ../httpreaderat`,
	}

	mod := importDetails{Base: "golang.org/x/mod", Version: "v0.31.0", Path: "."}
	modFile := importDetails{Base: "golang.org/x/mod", Version: "v0.31.0", Path: "modfile"}
	tools := importDetails{Base: "somewhere.org/tools", Version: "v0.1.0", Path: "."}
	httpreaderat := importDetails{Base: "../httpreaderat", Path: "."}
	httpreaderatsub := importDetails{Base: "../httpreaderat", Path: "sub"}

	if pkg, err := parseModFile(tfs, ""); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if im := pkg.Resolve("unknown.com/pkg"); im != nil {
		t.Errorf("expecting nil response, got %v", im)
	} else if im = pkg.Resolve("golang.org/x/mod"); im != nil && *im != mod {
		t.Errorf("expecting import %v, got %v", mod, im)
	} else if im = pkg.Resolve("golang.org/x/mod/modfile"); im != nil && *im != modFile {
		t.Errorf("expecting import %v, got %v", modFile, im)
	} else if im = pkg.Resolve("golang.org/x/tools"); im != nil && *im != tools {
		t.Errorf("expecting import %v, got %v", tools, im)
	} else if im = pkg.Resolve("vimagination.zapto.org/httpreaderat"); im != nil && *im != httpreaderat {
		t.Errorf("expecting import %v, got %v", httpreaderat, im)
	} else if im = pkg.Resolve("vimagination.zapto.org/httpreaderat/sub"); im != nil && *im != httpreaderatsub {
		t.Errorf("expecting import %v, got %v", httpreaderatsub, im)
	}
}

func TestAsFS(t *testing.T) {
	modFile := importDetails{Base: "golang.org/x/mod", Version: "v0.31.0", Path: "modfile"}
	cache := importDetails{Base: "vimagination.zapto.org/cache", Version: "v1.0.0", Path: "."}

	if f, err := modFile.AsFS(); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if _, err := f.OpenFile("print.go"); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if _, err := f.OpenFile("not-a-file.go"); err == nil {
		t.Error("expecting error, got nil")
	} else if _, ok := f.(*zipFS); ok {
		t.Log("was expecting FS to be a os.DirFS")
	}

	if f, err := cache.AsFS(); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if mf, err := parseModFile(f, ""); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if mf.Module != cache.Base {
		t.Errorf("expecting path %q, got %q", cache.Base, mf.Module)
	} else if _, ok := f.(*zipFS); !ok {
		t.Log("was expecting FS to be a zipFS")
	}
}

func TestTypes(t *testing.T) {
	pkg, err := ParsePackage(".")
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	obj := pkg.Scope().Lookup("moduleDetails")
	if obj == nil {
		t.Fatal("expecting object, got nil")
	}

	str, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		t.Fatal("expecting struct type")
	}

	typ := reflect.TypeOf(moduleDetails{})

	if str.NumFields() != typ.NumField() {
		t.Errorf("expecting %d fields, got %d", typ.NumField(), str.NumFields())
	}

	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0700); err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "sub", "a.go"), []byte("package subpkg\n\ntype A struct{\n\tB int\n}"), 0600); err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	if _, err := ParsePackage(filepath.Join(dir, "sub")); !errors.Is(err, errNoModFile) {
		t.Errorf("expecting error %q, got: %v", errNoModFile, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module vimagination.zapto.org/somepkg\n\ngo 1.25.5"), 0600); err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	if pkg, err := ParsePackage(filepath.Join(dir, "sub")); err != nil {
		t.Errorf("unexpected error: %#v", err)
	} else if name := pkg.Name(); name != "subpkg" {
		t.Errorf("expecting package name %q, got %q", "subpkg", name)
	} else if obj = pkg.Scope().Lookup("A"); obj == nil {
		t.Errorf("expecting object, got nil")
	}
}
