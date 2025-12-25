package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"testing"
)

var _ filesystem = &zipFS{}

func TestZipFS(t *testing.T) {
	var buf bytes.Buffer

	zw := zip.NewWriter(&buf)

	w, _ := zw.Create("package/a.txt")
	w.Write([]byte("12345"))

	w, _ = zw.Create("package/b.txt")
	w.Write([]byte("abcdefgh"))

	zw.Close()

	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	z := &zipFS{zr, ""}

	if entries, err := z.ReadDir("not-a-dir"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expecting error ErrNotExist, got %v", err)
	} else if entries != nil {
		t.Errorf("expecting nil entries, got %v", entries)
	} else if entries, err := z.ReadDir("."); err != nil {
		t.Errorf("expecting nil error, got %s", err)
	} else if len(entries) != 1 {
		t.Errorf("expecting 1 entry, got %d", len(entries))
	} else if n := entries[0].Name(); n != "package" {
		t.Errorf("expecting entry name %q, got %q", "package", n)
	} else if entries, _ := z.ReadDir("package"); len(entries) != 2 {
		t.Errorf("expecting 2 entries, got %d", len(entries))
	} else if n := entries[0].Name(); n != "a.txt" {
		t.Errorf("expecting entry name %q, got %q", "a.txt", n)
	} else if n := entries[1].Name(); n != "b.txt" {
		t.Errorf("expecting entry name %q, got %q", "b.txt", n)
	}

	if id := z.IsDir("not-a-file"); id {
		t.Errorf("expecting IsDir = false, got %v", id)
	} else if id = z.IsDir("package/"); !id {
		t.Errorf("expecting IsDir = true, got %v", id)
	}

	if data, err := z.ReadFile("not-a-file"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expecting error ErrNotExists, got %v", err)
	} else if data != nil {
		t.Errorf("expecting nil data, got %v", data)
	} else if data, err = z.ReadFile("package/a.txt"); err != nil {
		t.Errorf("expecting nil error, got %s", err)
	} else if string(data) != "12345" {
		t.Errorf("expecting to read %q, got %q", "12345", data)
	} else if data, err = z.ReadFile("package/b.txt"); err != nil {
		t.Errorf("expecting nil error, got %s", err)
	} else if string(data) != "abcdefgh" {
		t.Errorf("expecting to read %q, got %q", "abcdefgh", data)
	}
}
