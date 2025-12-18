package main

import (
	"archive/zip"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

type zipFS struct {
	*zip.Reader
}

func (z *zipFS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := z.Open(name)
	if err != nil {
		return nil, err
	}

	if d, ok := f.(fs.ReadDirFile); ok {
		return d.ReadDir(-1)
	}

	return nil, fs.ErrInvalid
}

type zipDir string

func (z zipDir) Name() string     { return filepath.Base(string(z)) }
func (zipDir) Size() int64        { return 0 }
func (zipDir) Mode() fs.FileMode  { return fs.ModeDir | fs.ModePerm }
func (zipDir) ModTime() time.Time { return time.Now() }
func (zipDir) IsDir() bool        { return true }
func (z zipDir) Sys() any         { return z }

func (z *zipFS) Stat(path string) (fs.FileInfo, error) {
	pathWithSlash := path

	if !strings.HasSuffix(path, "/") {
		pathWithSlash += "/"
	}

	for _, f := range z.File {
		if f.Name == path {
			return f.FileInfo(), nil
		} else if strings.HasPrefix(f.Name, pathWithSlash) {
			return zipDir(path), nil
		}
	}

	return nil, fs.ErrNotExist
}
