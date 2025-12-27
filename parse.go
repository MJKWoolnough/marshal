package main

import (
	"archive/zip"
	"errors"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"vimagination.zapto.org/httpreaderat"
)

type filesystem interface {
	OpenFile(string) (io.ReadCloser, error)
	IsDir(string) bool
	ReadDir(string) ([]fs.FileInfo, error)
	ReadFile(name string) ([]byte, error)
}

func ParsePackage(modulePath string, ignore ...string) (*types.Package, error) {
	var (
		m  *moduleDetails
		sd string
	)

	for path, sub := range splitPath(modulePath) {
		var err error

		if m, err = parseModFile(&osFS{os.DirFS(path).(statReadDirFileFS)}, path); err == nil {
			sd = sub

			break
		}
	}

	if m == nil {
		return nil, errNoModFile
	}

	return m.importPath(path.Join(m.Module, sd), ignore...)
}

func splitPath(path string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		left := strings.TrimSuffix(path, "/")
		right := ""

		for {
			if !yield(left, right) {
				return
			}

			pos := strings.LastIndexByte(left, '/')
			if pos == -1 {
				return
			}

			left = path[:pos]
			right = path[pos+1:]
		}
	}
}

func hasSubdir(root, dir string) (string, bool) {
	if strings.HasPrefix(dir, root) {
		return strings.TrimPrefix(dir, root), true
	}

	return "", false
}

func listGoFiles(fsys filesystem) ([]string, error) {
	ctx := build.Context{
		GOARCH:    runtime.GOARCH,
		GOOS:      runtime.GOOS,
		Compiler:  runtime.Compiler,
		IsDir:     fsys.IsDir,
		HasSubdir: hasSubdir,
		ReadDir:   fsys.ReadDir,
		OpenFile:  fsys.OpenFile,
	}

	pkg, err := ctx.ImportDir(".", 0)
	if err != nil {
		return nil, err
	}

	return pkg.GoFiles, nil
}

type moduleDetails struct {
	Module          string
	Path            string
	Imports         map[string]module.Version
	fset            *token.FileSet
	defaultImporter types.Importer
	cache           map[string]*types.Package
}

func parseModFile(fsys filesystem, path string) (*moduleDetails, error) {
	data, err := fsys.ReadFile("go.mod")
	if err != nil {
		return nil, err
	}

	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}

	imports := make(map[string]module.Version, len(f.Require))

	for _, r := range f.Require {
		imports[r.Mod.Path] = r.Mod
	}

	for _, r := range f.Replace {
		if m, ok := imports[r.Old.Path]; ok && (r.Old.Version == "" || r.Old.Version == m.Version) {
			imports[r.Old.Path] = r.New
		}
	}

	fset := token.NewFileSet()

	return &moduleDetails{
		Module:          f.Module.Mod.Path,
		Path:            path,
		Imports:         imports,
		fset:            fset,
		defaultImporter: importer.ForCompiler(fset, runtime.Compiler, nil),
		cache:           make(map[string]*types.Package),
	}, nil
}

func (m *moduleDetails) ParsePackage(fsys filesystem, pkgPath string, ignore ...string) (*types.Package, error) {
	files, err := listGoFiles(fsys)
	if err != nil {
		return nil, err
	}

	if len(ignore) > 0 {
		filtered := make([]string, 0, len(files))

		for _, file := range files {
			if !slices.Contains(ignore, file) {
				filtered = append(filtered, file)
			}
		}

		files = filtered
	}

	parsedFiles, err := m.parseFiles(pkgPath, fsys, files)
	if err != nil {
		return nil, err
	}

	var (
		conf = types.Config{
			GoVersion: runtime.Version(),
			Importer:  m,
		}
		info = types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}
	)

	return conf.Check(pkgPath, m.fset, parsedFiles, &info)
}

func (m *moduleDetails) parseFiles(pkgPath string, fsys filesystem, files []string) ([]*ast.File, error) {
	var pkg string

	parsedFiles := make([]*ast.File, len(files))

	for n, file := range files {
		f, err := fsys.OpenFile(file)
		if err != nil {
			return nil, err
		}

		pf, err := parser.ParseFile(m.fset, path.Join(pkgPath, file), f, parser.AllErrors|parser.ParseComments)
		if err != nil {
			return nil, err
		}

		if pkg == "" {
			pkg = pf.Name.Name
		} else if pkg != pf.Name.Name {
			return nil, errMultiplePackages
		}

		parsedFiles[n] = pf
	}

	return parsedFiles, nil
}

func (m *moduleDetails) Import(path string) (*types.Package, error) {
	if pkg, ok := m.cache[path]; ok {
		return pkg, nil
	}

	pkg, err := m.importPath(path)
	if err != nil {
		return nil, err
	}

	m.cache[path] = pkg

	return pkg, nil
}

func (m *moduleDetails) importPath(path string, ignore ...string) (*types.Package, error) {
	im := m.Resolve(path)
	if im == nil {
		return m.defaultImporter.Import(path)
	}

	fs, err := im.AsFS()
	if err != nil {
		return nil, err
	}

	return m.ParsePackage(fs, path, ignore...)
}

type importDetails struct {
	Base, Version, Path string
}

func (m *moduleDetails) Resolve(importURL string) *importDetails {
	if strings.HasPrefix(importURL, m.Module+"/") || importURL == m.Module {
		return &importDetails{Base: m.Path, Version: "", Path: strings.TrimPrefix(strings.TrimPrefix(importURL, m.Module), "/")}
	}

	for url, mod := range m.Imports {
		if url == importURL {
			return &importDetails{Base: mod.Path, Version: mod.Version, Path: "."}
		} else if strings.HasPrefix(importURL, url) {
			base := strings.TrimPrefix(importURL, url)

			if strings.HasPrefix(base, "/") {
				return &importDetails{Base: mod.Path, Version: mod.Version, Path: strings.TrimPrefix(base, "/")}
			}
		}
	}

	return nil
}

func (i *importDetails) CachedModPath() (string, error) {
	if modfile.IsDirectoryPath(i.Base) {
		return i.Base, nil
	}

	dir, err := i.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(build.Default.GOPATH, "pkg", "mod", dir), nil
}

func (i *importDetails) ModCacheURL() (string, error) {
	p, err := module.EscapePath(i.Base)
	if err != nil {
		return "", err
	}

	ver, err := module.EscapeVersion(i.Version)
	if err != nil {
		return "", err
	}

	return "https://proxy.golang.org" + path.Join("/", p, "@v", ver+".zip"), nil
}

func (i *importDetails) Dir() (string, error) {
	path, err := module.EscapePath(i.Base)
	if err != nil {
		return "", err
	}

	ver, err := module.EscapeVersion(i.Version)
	if err != nil {
		return "", err
	}

	return path + "@" + ver, nil
}

func (i *importDetails) AsFS() (filesystem, error) {
	local, err := i.CachedModPath()
	if err != nil {
		return nil, err
	}

	if s, err := os.Stat(local); err == nil {
		if s.IsDir() {
			return &osFS{os.DirFS(filepath.Join(local, i.Path)).(statReadDirFileFS)}, nil
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	return i.remotePackageFS()
}

func (i *importDetails) remotePackageFS() (filesystem, error) {
	remote, err := i.ModCacheURL()
	if err != nil {
		return nil, err
	}

	r, err := httpreaderat.NewRequest(remote)
	if err != nil {
		return nil, err
	}

	z, err := zip.NewReader(r, r.Length())
	if err != nil {
		return nil, err
	}

	dir, err := i.Dir()
	if err != nil {
		return nil, err
	}

	return &zipFS{z, dir}, nil
}

type statReadDirFileFS interface {
	fs.StatFS
	fs.ReadDirFS
	fs.ReadFileFS
}

type osFS struct {
	statReadDirFileFS
}

func (o *osFS) OpenFile(path string) (io.ReadCloser, error) {
	return o.Open(path)
}

func (o *osFS) IsDir(path string) bool {
	s, err := o.Stat(path)
	if err != nil {
		return false
	}

	return s.IsDir()
}

func (o *osFS) ReadDir(path string) ([]fs.FileInfo, error) {
	f, err := o.Open(path)
	if err != nil {
		return nil, err
	}

	return f.(*os.File).Readdir(-1)
}

var (
	errMultiplePackages = errors.New("multiple packages found")
	errNoModFile        = errors.New("no module file found")
)
