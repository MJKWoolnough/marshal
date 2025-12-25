package main

import (
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"time"
)

type testFile struct {
	name, contents string
}

func (t *testFile) Stat() (fs.FileInfo, error) { return t, nil }

func (t *testFile) Read(p []byte) (int, error) {
	n := copy(p, t.contents)

	t.contents = t.contents[n:]

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func (t testFile) Close() error { return nil }

func (t testFile) Name() string                { return t.name }
func (t testFile) Size() int64                 { return int64(len(t.contents)) }
func (t testFile) Mode() fs.FileMode           { return fs.ModeType }
func (t testFile) ModTime() time.Time          { return time.Now() }
func (t testFile) IsDir() bool                 { return false }
func (t *testFile) Sys() any                   { return t }
func (t *testFile) Type() fs.FileMode          { return fs.ModeType }
func (t *testFile) Info() (fs.FileInfo, error) { return t, nil }

type testDir string

func (t testDir) Name() string       { return string(t) }
func (t testDir) Size() int64        { return 0 }
func (t testDir) Mode() fs.FileMode  { return fs.ModeDir }
func (t testDir) ModTime() time.Time { return time.Now() }
func (t testDir) IsDir() bool        { return true }
func (t testDir) Sys() any           { return t }

type testFS map[string]string

func (t testFS) Open(name string) (fs.File, error) {
	contents, ok := t[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return &testFile{name: name, contents: contents}, nil
}

func (t testFS) Stat(name string) (fs.FileInfo, error) {
	if name == "." {
		return testDir("."), nil
	}

	contents, ok := t[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return &testFile{name: name, contents: contents}, nil
}

func (t testFS) ReadDir(dir string) ([]fs.DirEntry, error) {
	entries := make([]fs.DirEntry, 0, len(t))

	for name, contents := range t {
		entries = append(entries, &testFile{name: name, contents: contents})
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

	if pkg, err := m.ParsePackage("", tfs); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if a := pkg.Scope().Lookup("A"); a == nil {
		t.Error("expected type def, got nil")
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
	} else if _, err := f.Open("print.go"); err != nil {
		t.Errorf("unexpected error: %s", err)
	} else if _, err := f.Open("not-a-file.go"); err == nil {
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
	pkg, err := ParsePackage(os.DirFS(".").(filesystem), ".")
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
}
