package main

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
)

type AcLintArgs struct {
	Fn  string `json:"fn"`
	Src string `json:"src"`
}

type AcLintReport struct {
	Row int    `json:"row"`
	Col int    `json:"col"`
	Msg string `json:"msg"`
}

func init() {
	act(Action{
		Path: "/lint",
		Doc:  ``,
		Func: func(r Request) (data, error) {
			a := AcLintArgs{}
			res := make([]AcLintReport, 0)

			if err := r.Decode(&a); err != nil {
				return res, err
			}

			fset, af, err := parseAstFile(a.Fn, a.Src, parser.DeclarationErrors)
			if err == nil {
				res = lintCheckFlagParse(fset, af, res)
			} else if el, ok := err.(scanner.ErrorList); ok {
				for _, e := range el {
					res = append(res, AcLintReport{
						Row: e.Pos.Line - 1,
						Col: e.Pos.Column - 1,
						Msg: e.Msg,
					})
				}
			}

			return res, nil
		},
	})
}

func lintCheckFlagParse(fset *token.FileSet, af *ast.File, res []AcLintReport) []AcLintReport {
	ast.Inspect(af, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			if n.Name.String() == "main" || n.Name.String() == "main" {
				var flagCall *ast.CallExpr

				ast.Inspect(node, func(node ast.Node) bool {
					switch c := node.(type) {
					case *ast.CallExpr:
						if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
							if id, ok := sel.X.(*ast.Ident); ok && id.Name == "flag" {
								if sel.Sel.String() == "Parse" {
									flagCall = nil
								} else {
									flagCall = c
								}
							}
						}
					}
					return true
				})

				if flagCall != nil {
					tp := fset.Position(flagCall.Pos())
					s, _ := printSrc(fset, flagCall, true, 8)
					res = append(res, AcLintReport{
						Row: tp.Line - 1,
						Col: tp.Column - 1,
						Msg: "call " + s + " does not appear to have a corresponding call to flag.Parse()",
					})
				}

				return false
			}
		}

		return true
	})

	return res
}
