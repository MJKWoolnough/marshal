package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/MJKWoolnough/errors"
)

func e(explain string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", explain, err)
		os.Exit(1)
	}
}

func main() {
	var (
		pkgPath        string
		typeName       string
		outputFilename string
		json           bool
		bin            bool
	)
	flag.StringVar(&pkgPath, "p", ".", "package path")
	flag.BoolVar(&json, "j", false, "construct json (un)marshaler")
	flag.BoolVar(&bin, "b", false, "construct binary (un)marshaler")
	flag.StringVar(&typeName, "t", "", "type name")
	flag.StringVar(&outputFilename, "o", "", "output filename")
	flag.Parse()

	if typeName == "" {
		e("error", errors.Error("need type name"))
	} else if !json && !bin {
		e("error", errors.Error("need a format"))
	}

	wd, err := os.Getwd()
	e("error getting working directory", err)

	c := exec.Command("go", "test")
	c.Dir = filepath.Join(wd, pkgPath)

	stat, err := os.Stat(c.Dir)
	e("error getting directory information", err)

	if !stat.IsDir() {
		e("error", errors.Error("path is not a directory"))
	}

	e("error running `go test`", c.Run())

	fset := token.NewFileSet()
	p, err := parser.ParseDir(fset, pkgPath, nil, parser.AllErrors)
	e("error parsing go files", err)

	var (
		packageName string
		found       bool
	)
Loop:
	for _, pkg := range p {
		for filename, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch gd := decl.(type) {
				case *ast.GenDecl:
					if gd.Tok == token.TYPE {
						for _, spec := range gd.Specs {
							switch ts := spec.(type) {
							case *ast.TypeSpec:
								if ts.Name.Name == typeName {
									found = true
									packageName = file.Name.Name
									if outputFilename == "" {
										outputFilename = filepath.Join(c.Dir, filename[:len(filename)-3]+"_marshaler.go")
									}
									break Loop
								}
							}
						}
					}
				}
			}
		}
	}
	if !found {
		e("error", errors.Error("unable to determine file containing type"))
	}

	var options []byte

	if bin {
		options = append(options, ", marshal.Binary()"...)
	}
	if json {
		options = append(options, ", marshal.JSON()"...)
	}

	tmpGenFile := filepath.Join(c.Dir, "temp_marshall_generate_test.go")
	f, err := os.Create(tmpGenFile)
	e("error creating generator file", err)

	_, err = fmt.Fprintf(f, template, packageName, typeName, packageName, outputFilename, options)
	e("error writing generator file", err)
	e("error closing generator file", f.Close())

	ct := exec.Command("go", "test", "-run", "TestMarshalGenerator")
	ct.Dir = c.Dir
	err = ct.Run()
	e("error removing temporary file", os.Remove(tmpGenFile))
	e("error running `go test` (2)", err)
}

const template = `package %s

import (
	"testing"

	"github.com/MJKWoolnough/marshal"
)

func TestMarshalGenerator(t *testing.T) {
	if err := marshal.Generate((*%s)(nil), %q, %q%s); err != nil {
		t.Fatal(err)
	}
}
`
