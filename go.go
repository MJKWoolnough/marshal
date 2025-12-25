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
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"vimagination.zapto.org/httpreaderat"
)

type filesystem interface {
	fs.FS
	fs.StatFS
	fs.ReadDirFS
	fs.ReadFileFS
}

type dirEntry struct {
	fs.DirEntry
}

func ParsePackage(fsys filesystem, path string) (*types.Package, error) {
	m, err := parseModFile(fsys, path)
	if err != nil {
		return nil, err
	}

	return m.ParsePackage(m.Module, fsys)
}

func listGoFiles(fsys filesystem) ([]string, error) {
	ctx := build.Context{
		GOARCH:   runtime.GOARCH,
		GOOS:     runtime.GOOS,
		Compiler: runtime.Compiler,
		IsDir: func(path string) bool {
			s, err := fsys.Stat(path)
			if err != nil {
				return false
			}

			return s.IsDir()
		},
		HasSubdir: func(root, dir string) (string, bool) {
			if strings.HasPrefix(dir, root) {
				return strings.TrimPrefix(dir, root), true
			}

			return "", false
		},
		ReadDir: func(dir string) ([]fs.FileInfo, error) {
			entries, err := fsys.ReadDir(dir)
			if err != nil {
				return nil, err
			}

			fis := make([]fs.FileInfo, len(entries))

			for n, entry := range entries {
				fis[n], err = entry.Info()
				if err != nil {
					return nil, err
				}
			}

			return fis, nil
		},
		OpenFile: func(path string) (io.ReadCloser, error) {
			return fsys.Open(path)
		},
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

func (m *moduleDetails) ParsePackage(pkgPath string, fsys filesystem) (*types.Package, error) {
	files, err := listGoFiles(fsys)
	if err != nil {
		return nil, err
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

	return conf.Check(".", m.fset, parsedFiles, &info)
}

func (m *moduleDetails) parseFiles(pkgPath string, fsys filesystem, files []string) ([]*ast.File, error) {
	var pkg string

	parsedFiles := make([]*ast.File, len(files))

	for n, file := range files {
		f, err := fsys.Open(file)
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

func (m *moduleDetails) importPath(path string) (*types.Package, error) {
	im := m.Resolve(path)
	if im == nil {
		return m.defaultImporter.Import(path)
	}

	fs, err := im.AsFS()
	if err != nil {
		return nil, err
	}

	return m.ParsePackage(path, fs)
}

type importDetails struct {
	Base, Version, Path string
}

func (m *moduleDetails) Resolve(importURL string) *importDetails {
	if strings.HasPrefix(importURL, m.Module+"/") {
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
			return os.DirFS(filepath.Join(local, i.Path)).(filesystem), nil
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

var errMultiplePackages = errors.New("multiple packages found")
