package main

import (
	"archive/zip"
	"io/fs"
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
