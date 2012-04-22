package main

import (
	"bytes"
	"go/parser"
	"go/printer"
)

type AcFmtArgs struct {
	Fn  string `json:"fn"`
	Src string `json:"src"`
}

func init() {
	act(Action{
		Path: "/fmt",
		Doc: `
formats the source like gofmt does
@data: {"fn": "...", "src": "..."}
@resp: "formatted source"
`,
		Func: func(r Request) (data, error) {
			a := AcFmtArgs{}
			res := ""
			if err := r.Decode(&a); err != nil {
				return res, err
			}

			fset, af, err := parseAstFile(a.Fn, a.Src, parser.ParseComments)
			if err == nil {
				buf := &bytes.Buffer{}
				if err = printer.Fprint(buf, fset, af); err == nil {
					res = buf.String()
				}
			}
			return res, err
		},
	})
}
