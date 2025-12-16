package main

import (
	"io"
	"io/fs"
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
