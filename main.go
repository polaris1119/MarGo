package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
)

var (
	methods = map[string]Method{}
)

type Header struct {
	Method string `json:"method"`
}

type data interface{}

type Response struct {
	Error string `json:"error"`
	Data  data   `json:"data"`
}

type Request struct {
	*bufio.Reader
}

func (r Request) Decode(a interface{}) error {
	if err := json.NewDecoder(r).Decode(a); err != io.EOF {
		return err
	}
	return nil
}

func parseAstFile(fn string, s string, mode parser.Mode) (fset *token.FileSet, af *ast.File, err error) {
	fset = token.NewFileSet()
	var src interface{}
	if s != "" {
		src = s
	}
	if fn == "" {
		fn = "<stdin>"
	}
	af, err = parser.ParseFile(fset, fn, src, mode)
	return
}

type Method func(r Request) (data, error)

func respond(conn net.Conn) {
	h := Header{}
	r := Request{bufio.NewReader(conn)}
	resp := Response{}
	line, err := r.ReadSlice('\r')
	if err == nil {
		err = json.Unmarshal(line, &h)
		if err == nil {
			if h.Method != "exit" {
				meth := methods[h.Method]
				if meth == nil {
					err = errors.New("Invalid method call `" + h.Method + "`")
				} else {
					resp.Data, err = meth(r)
				}
			}
		}

		if err != nil {
			resp.Error = err.Error()
		}
	} else {
		resp.Error = "Invalid Request Header: " + err.Error()
	}

	json.NewEncoder(conn).Encode(resp)
	conn.Close()

	if h.Method == "exit" {
		os.Exit(0)
	}
}

func main() {
	d := flag.Bool("d", false, "Whether or not to launch in the background(like a daemon)")
	closeFds := flag.Bool("close-fds", false, "Whether or not to close stdin, stdout and stderr")
	addr := flag.String("addr", "127.9.5.1:57951", "The tcp address to listen on")
	flag.Parse()

	if *d {
		cmd := exec.Command(os.Args[0], "-close-fds", "-addr", *addr)
		serr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalln(err)
		}
		err = cmd.Start()
		if err != nil {
			log.Fatalln(err)
		}
		s, err := ioutil.ReadAll(serr)
		s = bytes.TrimSpace(s)
		if bytes.HasPrefix(s, []byte("addr: ")) {
			fmt.Println(string(s))
			cmd.Process.Release()
		} else {
			log.Printf("unexpected response from MarGo: `%s` error: `%v`\n", s, err)
			cmd.Process.Kill()
		}
	} else {
		l, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Fprintf(os.Stderr, "addr: %s\n", l.Addr())
		if *closeFds {
			os.Stdin.Close()
			os.Stdout.Close()
			os.Stderr.Close()
		}

		go func() {
			importPaths(map[string]string{})
		}()

		for {
			conn, err := l.Accept()
			if err != nil {
				if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
					log.Fatalln(err)
				}
			}
			go respond(conn)
		}
	}
}
