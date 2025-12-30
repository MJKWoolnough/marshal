package main

import (
	"go/ast"
	"go/format"
	"go/token"
	"io"
)

func constructFile(w io.Writer, pkg string) {
	fset := token.NewFileSet()
	file := &ast.File{
		Name: ast.NewIdent(pkg),
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"io"`,
						},
					},
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"vimagination.zapto.org/byteio"`,
						},
					},
				},
			},
		},
	}

	format.Node(w, fset, file)
}
