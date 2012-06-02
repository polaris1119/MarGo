package main

import (
	"bytes"
	"errors"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type Doc struct {
	Decl     string `json:"decl"`
	Synopsis string `json:"synopsis"`
	Name     string `json:"name"`
	Doc      string `json:"doc"`
	Fn       string `json:"fn"`
	Row      int    `json:"row"`
	Col      int    `json:"col"`
}

type DocArgs struct {
	Fn   string            `json:"fn"`
	Src  string            `json:"src"`
	Expr string            `json:"expr"`
	Env  map[string]string `json:"env"`
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

			parts := strings.Split(strings.TrimSpace(a.Expr), ".")
			if len(parts) < 2 {
				return res, errors.New("N/I: cannot decode expression: expect pkg.Func")
			}

			fset, af, err := parseAstFile(a.Fn, a.Src, parser.ImportsOnly)
			if err != nil {
				return res, err
			}

			pkgPath := ""
			pattern := parts[1]

			for _, im := range af.Imports {
				if im.Name != nil && parts[0] == im.Name.Name {
					pkgPath = im.Path.Value
					break
				}
			}
			if pkgPath == "" {
				for _, im := range af.Imports {
					imPath := unquote(im.Path.Value)
					if parts[0] == path.Base(imPath) {
						pkgPath = imPath
						break
					}
				}
			}

			if pkgPath == "" || pattern == "" {
				return res, errors.New("N/I: cannot find package import")
			}

			paths := map[string]bool{}
			env := []string{
				a.Env["GOPATH"],
				a.Env["GOROOT"],
				os.Getenv("GOPATH"),
				os.Getenv("GOROOT"),
				runtime.GOROOT(),
			}
			for _, ent := range env {
				for _, path := range filepath.SplitList(ent) {
					if path != "" {
						paths[filepath.Join(path, "src", "pkg")] = true
					}
				}
			}

			for root, _ := range paths {
				dpath := filepath.Join(root, pkgPath)
				st, err := os.Stat(dpath)
				if err == nil && st.IsDir() {
					docs := findDocs(fset, root, dpath, pkgPath, pattern)
					if len(docs) > 0 {
						res = docs
						break
					}
				}
			}

			return res, nil
		},
	})
}

func findDocs(fset *token.FileSet, root, dpath, importPath, pat string) (docs []*Doc) {
	pkgs, _ := parser.ParseDir(fset, dpath, nil, parser.ParseComments)
	buf := bytes.NewBuffer(nil)
	for _, pkg := range pkgs {
		pd := doc.New(pkg, importPath, 0)
		for _, v := range pd.Funcs {
			if pat == v.Name {
				tp := fset.Position(v.Decl.Pos())
				buf.Reset()
				printer.Fprint(buf, fset, v.Decl.Type)
				docs = append(docs, &Doc{
					Decl:     buf.String(),
					Name:     v.Name,
					Doc:      v.Doc,
					Synopsis: doc.Synopsis(v.Doc),
					Fn:       tp.Filename,
					Row:      tp.Line - 1,
					Col:      tp.Column - 1,
				})
			}
		}
	}
	return
}
