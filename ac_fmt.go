package main

import (
	"bytes"
	"go/parser"
	"go/printer"
)

type AcFmtArgs struct {
	Fn        string `json:"fn"`
	Src       string `json:"src"`
	TabIndent bool   `json:"tab_indent"`
	TabWidth  int    `json:"tab_width"`
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
			a := AcFmtArgs{
				TabIndent: true,
				TabWidth:  8,
			}

			res := ""
			if err := r.Decode(&a); err != nil {
				return res, err
			}

			fset, af, err := parseAstFile(a.Fn, a.Src, parser.ParseComments)
			if err == nil {
				buf := &bytes.Buffer{}
				mode := printer.UseSpaces
				if a.TabIndent {
					mode |= printer.TabIndent
				}
				p := &printer.Config{
					Mode:     mode,
					Tabwidth: a.TabWidth,
				}
				if err = p.Fprint(buf, fset, af); err == nil {
					res = buf.String()
				}
			}
			return res, err
		},
	})
}
