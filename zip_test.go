package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"testing"
)

func TestZipFS(t *testing.T) {
	var buf bytes.Buffer

	zw := zip.NewWriter(&buf)

	w, _ := zw.Create("package/a.txt")
	w.Write([]byte("12345"))

	w, _ = zw.Create("package/b.txt")
	w.Write([]byte("abcdefgh"))

	zw.Close()

	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	z := &zipFS{zr}

	if entries, _ := z.ReadDir("."); len(entries) != 1 {
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

	if stat, err := z.Stat("not-a-file"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expecting error ErrNotExist, got %v", err)
	} else if stat != nil {
		t.Errorf("expecting nil stat, got %v", stat)
	} else if stat, err = z.Stat("package/"); err != nil {
		t.Errorf("expecting nil error, got %s", err)
	} else if !stat.IsDir() {
		t.Error("expecting IsDir() to be true")
	} else if stat, err = z.Stat("package/a.txt"); err != nil {
		t.Errorf("expecting nil error, got %s", err)
	} else if size := stat.Size(); size != 5 {
		t.Errorf("expecting size 5, got %d", size)
	}
}
