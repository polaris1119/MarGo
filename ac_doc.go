package main

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
)

type Doc struct {
	Decl     string `json:"decl"`
	Src      string `json:"src"`
	Synopsis string `json:"synopsis"`
	Name     string `json:"name"`
	Doc      string `json:"doc"`
	Fn       string `json:"fn"`
	Row      int    `json:"row"`
	Col      int    `json:"col"`
}

type DocArgs struct {
	Fn        string            `json:"fn"`
	Src       string            `json:"src"`
	Expr      string            `json:"expr"`
	Env       map[string]string `json:"env"`
	Offset    int               `json:"offset"`
	TabIndent bool              `json:"tab_indent"`
	TabWidth  int               `json:"tab_width"`
}

func init() {
	act(Action{
		Path: "/doc",
		Doc:  "",
		Func: func(r Request) (data, error) {
			res := []*Doc{}

			a := DocArgs{Env: map[string]string{}}

			if err := r.Decode(&a); err != nil {
				return res, err
			}

			fset, af, err := parseAstFile(a.Fn, a.Src, parser.ParseComments)
			if err != nil {
				return res, err
			}

			sel, id := identAt(fset, af, a.Offset)

			pkg, err := parsePkg(fset, filepath.Dir(a.Fn), parser.ParseComments)
			if pkg == nil {
				return nil, err
			}

			obj := findUnderlyingObj(fset, af, pkg, rootDirs(a.Env), sel, id)
			if obj != nil {
				declSrc := "" // todo: print the declaration
				objSrc, _ := printSrc(fset, obj.Decl, a.TabIndent, a.TabWidth)
				docText := "no docs found for " + obj.Kind.String()
				switch v := obj.Decl.(type) {
				case *ast.FuncDecl:
					docText = v.Doc.Text()
				default:
					// todo: collect doc for other types as well
				}

				tp := fset.Position(obj.Pos())
				res = append(res, &Doc{
					Decl:     declSrc,
					Src:      objSrc,
					Name:     obj.Name,
					Doc:      docText,
					Synopsis: doc.Synopsis(docText),
					Fn:       tp.Filename,
					Row:      tp.Line - 1,
					Col:      tp.Column - 1,
				})
			}
			return res, nil
		},
	})
}

func isBetween(n, start, end int) bool {
	return (n >= start && n <= end)
}

func identAt(fset *token.FileSet, af *ast.File, offset int) (sel *ast.SelectorExpr, id *ast.Ident) {
	ast.Inspect(af, func(n ast.Node) bool {
		if n != nil {
			start := fset.Position(n.Pos())
			end := fset.Position(n.End())
			if isBetween(offset, start.Offset, end.Offset) {
				switch v := n.(type) {
				case *ast.SelectorExpr:
					sel = v
				case *ast.Ident:
					id = v
				}
			}
		}
		return true
	})
	return
}

func findUnderlyingObj(fset *token.FileSet, af *ast.File, pkg *ast.Package, srcRootDirs []string, sel *ast.SelectorExpr, id *ast.Ident) *ast.Object {
	if id != nil && id.Obj != nil {
		return id.Obj
	}

	if id == nil {
		// can this ever happen?
		return nil
	}

	if sel == nil {
		if obj := pkg.Scope.Lookup(id.Name); obj != nil {
			return obj
		}
	}

	if sel == nil {
		return nil
	}

	switch x := sel.X.(type) {
	case *ast.Ident:
		if x.Obj != nil {
			// todo: resolve type
		} else {
			if v := pkg.Scope.Lookup(id.Name); v != nil {
				// todo: found a type?
			} else {
				// it's most likely a package
				// todo: handle .dot imports
				for _, ispec := range af.Imports {
					importPath := unquote(ispec.Path.Value)
					pkgAlias := ""
					if ispec.Name == nil {
						pkgAlias = path.Base(importPath)
					} else {
						pkgAlias = ispec.Name.Name
					}
					if pkgAlias == x.Name {
						if id == x {
							// where do we go as the first place of a package?
							return nil
						}

						if p, _ := findPkg(fset, importPath, srcRootDirs, parser.ParseComments); p != nil {
							if obj := p.Scope.Lookup(id.Name); obj != nil {
								return obj
							}
							return nil
						}
					}
				}
			}
		}
	}
	return nil
}
