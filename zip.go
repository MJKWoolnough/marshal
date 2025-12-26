package main

import (
	"archive/zip"
	"io"
	"io/fs"
	"path"
	"strings"
)

type zipFS struct {
	*zip.Reader
	base string
}

func (z *zipFS) OpenFile(name string) (io.ReadCloser, error) {
	return z.Open(path.Join(z.base, name))
}

func (z *zipFS) ReadDir(name string) ([]fs.FileInfo, error) {
	f, err := z.Open(path.Join(z.base, name))
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
	f, err := z.Open(path.Join(z.base, name))
	if err != nil {
		return nil, err
	}

	s, _ := f.Stat()

	buf := make([]byte, s.Size())

	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, err
	}

	return buf, nil
}

func (z *zipFS) IsDir(dir string) bool {
	dir = path.Join(z.base, dir)
	pathWithSlash := dir

	if !strings.HasSuffix(dir, "/") {
		pathWithSlash += "/"
	}

	for _, f := range z.File {
		if f.Name == dir {
			return false
		} else if strings.HasPrefix(f.Name, pathWithSlash) {
			return true
		}
	}

	return false
}
