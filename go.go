package main

import (
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
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

func ListGoFiles(fsys filesystem) ([]string, error) {
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

func ParsePackage(fsys filesystem) (*types.Package, error) {
	files, err := ListGoFiles(fsys)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	var pkg string

	parsedFiles := make([]*ast.File, len(files))

	for n, file := range files {
		f, err := fsys.Open(file)
		if err != nil {
			return nil, err
		}

		pf, err := parser.ParseFile(fset, file, f, parser.AllErrors|parser.ParseComments)
		if err != nil {
			return nil, err
		}

		if pkg == "" {
			pkg = pf.Name.Name
		} else if pkg != pf.Name.Name {
			return nil, errors.New("multiple packages found")
		}

		parsedFiles[n] = pf
	}

	var (
		conf types.Config
		info = types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}
	)

	return conf.Check(".", fset, parsedFiles, &info)
}

type Module struct {
	Path    string
	Imports map[string]string
}

func ParseModFile(fsys filesystem) (*Module, error) {
	data, err := fsys.ReadFile("go.mod")
	if err != nil {
		return nil, err
	}

	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}

	imports := make(map[string]string, len(f.Require))

	for _, r := range f.Require {
		imports[r.Mod.Path] = r.Mod.Version
	}

	return &Module{
		Path:    f.Module.Mod.Path,
		Imports: imports,
	}, nil
}

func CachedModPath(pkg, version string) (string, error) {
	path, err := module.EscapePath(pkg)
	if err != nil {
		return "", err
	}

	ver, err := module.EscapeVersion(version)
	if err != nil {
		return "", err
	}

	return filepath.Join(build.Default.GOPATH, "pkg", "mod", path+"@"+ver), nil
}

func ModCacheURL(pkg, version string) (string, error) {
	p, err := module.EscapePath(pkg)
	if err != nil {
		return "", err
	}

	ver, err := module.EscapeVersion(version)
	if err != nil {
		return "", err
	}

	return "https://proxy.golang.org" + path.Join("/", p, "@v", ver+".zip"), nil
}
