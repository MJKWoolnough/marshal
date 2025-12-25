package main

import (
	"archive/zip"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

type zipFS struct {
	*zip.Reader
	base string
}

func (z *zipFS) OpenFile(name string) (io.ReadCloser, error) {
	return z.Open(filepath.Join(z.base, name))
}

func (z *zipFS) ReadDir(name string) ([]fs.FileInfo, error) {
	f, err := z.Open(filepath.Join(z.base, name))
	if err != nil {
		return nil, err
	}

	d, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, fs.ErrInvalid
	}

	entries, err := d.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	fis := make([]fs.FileInfo, len(entries))

	for n, entry := range entries {
		fis[n] = entry.(fs.FileInfo)
	}

	return fis, nil
}

func (z *zipFS) ReadFile(name string) ([]byte, error) {
	f, err := z.Open(filepath.Join(z.base, name))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(f)
}

func (z *zipFS) IsDir(path string) bool {
	path = filepath.Join(z.base, path)
	pathWithSlash := path

	if !strings.HasSuffix(path, "/") {
		pathWithSlash += "/"
	}

	for _, f := range z.File {
		if f.Name == path {
			return false
		} else if strings.HasPrefix(f.Name, pathWithSlash) {
			return true
		}
	}

	return false
}
