package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

type NoInputErr string

func (s NoInputErr) Error() string {
	return string(s)
}

var (
	actions    = map[string]Action{}
	acLck      = sync.Mutex{}
	acQuitting = false
	acWg       = sync.WaitGroup{}
	acListener net.Listener
)

func act(ac Action) {
	ac.Path = normPath(ac.Path)
	acLck.Lock()
	defer acLck.Unlock()
	if _, exists := actions[ac.Path]; exists {
		log.Fatalf("Action exists: %s\n", ac.Path)
	}
	if ac.Func == nil {
		log.Fatalf("Invalid action: %s\n", ac.Path)
	}
	actions[ac.Path] = ac
}

func normPath(p string) string {
	return path.Clean("/" + strings.ToLower(strings.TrimSpace(p)))
}

type data interface{}

type Response struct {
	Error string `json:"error"`
	Data  data   `json:"data"`
}

type Request struct {
	Rw  http.ResponseWriter
	Req *http.Request
}

func (r Request) Decode(a interface{}) error {
	data := []byte(r.Req.FormValue("data"))
	if len(data) == 0 {
		return NoInputErr("Data is empty")
	}
	return json.Unmarshal(data, a)
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

type Action struct {
	Path string
	Doc  string
	Func func(r Request) (data, error)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func serve(rw http.ResponseWriter, req *http.Request) {
	acWg.Add(1)
	defer acWg.Done()

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	r := Request{
		Rw:  rw,
		Req: req,
	}
	path := normPath(req.URL.Path)
	resp := Response{}

	defer func() {
		json.NewEncoder(rw).Encode(resp)
	}()

	if ac, ok := actions[path]; ok {
		var err error
		resp.Data, err = ac.Func(r)
		if err != nil {
			resp.Error = err.Error()
		}
	} else {
		resp.Error = "Invalid action: " + path
	}
}

func main() {
	d := flag.Bool("d", false, "Whether or not to launch in the background(like a daemon)")
	closeFds := flag.Bool("close-fds", false, "Whether or not to close stdin, stdout and stderr")
	addr := flag.String("addr", "127.0.0.1:57951", "The tcp address to listen on")
	quit := flag.Bool("quit", false, "Request that an existing instance of MarGo listening on *addr* quit")
	replace := flag.Bool("replace", false, "Request that an existing instance of MarGo listening on *addr* quit and then start listening on *addr*")
	flag.Parse()

	if *quit || *replace {
		if resp, err := http.Get(`http://` + *addr + `/?data="bye%20ni"`); err == nil {
			resp.Body.Close()
		}
		if *quit {
			return
		}
	}

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
		var err error
		acListener, err = net.Listen("tcp", *addr)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Fprintf(os.Stderr, "addr: http://%s\n", acListener.Addr())
		if *closeFds {
			os.Stdin.Close()
			os.Stdout.Close()
			os.Stderr.Close()
		}

		go func() {
			importPaths(map[string]string{})
		}()

		err = http.Serve(acListener, http.HandlerFunc(serve))
		if !acQuitting && err != nil {
			log.Fatalln(err)
		}
		acWg.Wait()
	}
}
