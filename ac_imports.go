package main

import (
	"bytes"
	"fmt"
	"go/ast" // parse...
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

/* comment */

type ImportDecl struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type ImportsResult struct {
	FileImports  []ImportDecl `json:"file_imports"`
	ImportPaths  []string     `json:"import_paths"`
	Src          string       `json:"src"`
	SizeRef      int          `json:"size_ref"`
	LineRef      int          `json:"line_ref"`
	LineCountRef int          `json:"line_count_ref"`
}

type ImportsArgs struct {
	Fn          string            `json:"fn"`
	Src         string            `json:"src"`
	ImportPaths bool              `json:"import_paths"`
	Toggle      []ImportDecl      `json:"toggle"`
	Env         map[string]string `json:"env"`
}

func unquote(s string) string {
	return strings.Trim(s, "\"`")
}

func quote(s string) string {
	return `"` + unquote(s) + `"`
}

func init() {
	act(Action{
		Path: "/imports",
		Doc:  "",
		Func: func(r Request) (data, error) {
			res := ImportsResult{
				FileImports: []ImportDecl{},
				ImportPaths: []string{},
			}

			a := ImportsArgs{
				Toggle: []ImportDecl{},
				Env:    map[string]string{},
			}

			if err := r.Decode(&a); err != nil {
				return res, err
			}

			if a.ImportPaths {
				res.ImportPaths, _ = importPaths(a.Env)
			}

			fset, af, err := parseAstFile(a.Fn, a.Src, parser.ImportsOnly|parser.ParseComments)
			if err == nil {
				tf := fset.File(af.Pos())
				res.LineCountRef = tf.LineCount()

				for _, v := range af.Decls {
					if p := fset.Position(v.End()); p.IsValid() {
						res.LineRef = maxInt(res.LineRef, p.Line)
						res.SizeRef = maxInt(res.SizeRef, p.Offset)
					}
				}
				for _, v := range af.Comments {
					if p := fset.Position(v.End()); p.IsValid() {
						res.LineRef = maxInt(res.LineRef, p.Line)
						res.SizeRef = maxInt(res.SizeRef, p.Offset)
					}
				}

				toggle := map[ImportDecl]bool{}
				for _, v := range a.Toggle {
					toggle[v] = true
				}

				var first *ast.GenDecl
				for j := 0; j < len(af.Decls); j += 1 {
					d := af.Decls[j]
					if decl, ok := d.(*ast.GenDecl); ok {
						if decl.Tok == token.IMPORT {
							for i := 0; i < len(decl.Specs); i += 1 {
								if sp, ok := decl.Specs[i].(*ast.ImportSpec); ok {
									id := ImportDecl{Path: unquote(sp.Path.Value)}
									if sp.Name != nil {
										id.Name = sp.Name.String()
									}
									if _, ok := toggle[id]; ok {
										delete(toggle, id)
										decl.Specs = append(decl.Specs[:i], decl.Specs[i+1:]...)
									} else {
										res.FileImports = append(res.FileImports, id)
									}
								}
							}
							if len(decl.Specs) == 0 && decl.Lparen == token.NoPos {
								af.Decls = append(af.Decls[:j], af.Decls[j+1:]...)
							} else if first == nil {
								first = decl
							}
						}
					}
				}

				if len(toggle) > 0 {
					if first == nil {
						first = &ast.GenDecl{
							Tok:    token.IMPORT,
							Lparen: 1,
						}
						af.Decls = append(af.Decls, first)
					} else if first.Lparen == token.NoPos {
						first.Lparen = 1
					}
					for id, _ := range toggle {
						sp := &ast.ImportSpec{
							Path: &ast.BasicLit{
								Value: quote(id.Path),
								Kind:  token.STRING,
							},
						}
						first.Specs = append(first.Specs, sp)
						res.FileImports = append(res.FileImports, id)
					}
				}
				ast.SortImports(fset, af)

				if len(a.Toggle) > 0 {
					buf := &bytes.Buffer{}
					if err = printer.Fprint(buf, fset, af); err == nil {
						res.Src = buf.String()
					}
				}
			}
			return res, err
		},
	})
}
