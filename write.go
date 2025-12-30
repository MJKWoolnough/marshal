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
							Kind:     token.STRING,
							Value:    `"vimagination.zapto.org/byteio"`,
							ValuePos: 3,
						},
					},
				},
			},
		},
	}

	wsfile := fset.AddFile("out.go", 1, 4)

	wsfile.SetLines([]int{0, 1, 2, 3})
	format.Node(w, fset, file)
}
