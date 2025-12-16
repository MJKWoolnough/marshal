package main

import (
	"go/build"
	"io"
	"io/fs"
	"runtime"
	"strings"
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
