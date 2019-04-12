package gocredits

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"unicode/utf8"
)

const (
	cmdName     = "gocredits"
	defaultTmpl = `{{range $_, $elm := .Licenses -}}
{{$elm.Name}}
{{$elm.URL}}
----------------------------------------------------------------
{{$elm.Content}}
================================================================

{{end}}`
)

// Run the gocredits
func Run(argv []string, outStream, errStream io.Writer) error {
	log.SetOutput(errStream)
	fs := flag.NewFlagSet(
		fmt.Sprintf("%s (v%s rev:%s)", cmdName, version, revision), flag.ContinueOnError)
	fs.SetOutput(errStream)
	ver := fs.Bool("version", false, "display version")
	var (
		format    = fs.String("f", "", "format")
		write     = fs.Bool("w", false, "write result to CREDITS file instead of stdout")
		printJSON = fs.Bool("json", false, "data to be printed in JSON format")
	)
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if *ver {
		return printVersion(outStream)
	}
	modPath := fs.Arg(0)
	if modPath == "" {
		modPath = "."
	}
	licenses, err := takeCredits(modPath)
	if err != nil {
		return err
	}
	data := struct {
		Licenses []*license
	}{
		Licenses: licenses,
	}
	if *printJSON {
		return json.NewEncoder(outStream).Encode(data)
	}

	tmplStr := *format
	if tmplStr == "" {
		tmplStr = defaultTmpl
	}
	tmpl, err := template.New(cmdName).Parse(tmplStr)
	if err != nil {
		return err
	}
	out := outStream
	if *write {
		f, err := os.OpenFile(filepath.Join(modPath, "CREDITS"), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	return tmpl.Execute(out, data)
}

func printVersion(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%s v%s (rev:%s)\n", cmdName, version, revision)
	return err
}

type license struct {
	Name, URL, FilePath, Content string
}

func takeCredits(dir string) ([]*license, error) {
	goroot := runtime.GOROOT()
	var (
		err   error
		bs    []byte
		lpath string
	)
	for _, lpath = range []string{"LICENSE", "../LICENSE"} {
		bs, err = ioutil.ReadFile(filepath.Join(goroot, lpath))
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	ret := []*license{
		&license{
			Name:     "Go (the standard library)",
			URL:      "https://golang.org/",
			FilePath: lpath,
			Content:  string(bs),
		},
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	gopkgmod := filepath.Join(gopath, "pkg", "mod")

	gosum := filepath.Join(dir, "go.sum")
	f, err := os.Open(gosum)
	if err != nil {
		if os.IsNotExist(err) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
				return nil, fmt.Errorf("use go modules")
			}
			return ret, nil
		}
		return nil, err
	}
	defer f.Close()

	scr := bufio.NewScanner(f)
	for scr.Scan() {
		stuff := strings.Fields(scr.Text())
		if len(stuff) != 3 {
			continue
		}
		if strings.HasSuffix(stuff[1], "/go.mod") {
			continue
		}
		encodedPath, err := encodeString(stuff[0])
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(gopkgmod, encodedPath+"@"+stuff[1])
		licenseFile, content, err := findLicense(dir)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &license{
			Name:     stuff[0],
			URL:      fmt.Sprintf("https://%s", stuff[0]),
			FilePath: filepath.Join(dir, licenseFile),
			Content:  content,
		})
	}
	if err := scr.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func findLicense(dir string) (string, string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", "", err
	}
	var (
		bestScore = 0.0
		fileName  = ""
	)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		n := f.Name()
		score := scoreLicenseName(n)
		if score > bestScore {
			bestScore = score
			fileName = n
		}
	}
	if fileName == "" {
		return "", "", fmt.Errorf("no LICENSE files found in %q", dir)
	}
	bs, err := ioutil.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return "", "", err
	}
	return fileName, string(bs), nil
}

// copied from cmd/go/internal/module/module.go
func encodeString(s string) (encoding string, err error) {
	haveUpper := false
	for _, r := range s {
		if r == '!' || r >= utf8.RuneSelf {
			// This should be disallowed by CheckPath, but diagnose anyway.
			// The correctness of the encoding loop below depends on it.
			return "", fmt.Errorf("internal error: inconsistency in EncodePath")
		}
		if 'A' <= r && r <= 'Z' {
			haveUpper = true
		}
	}

	if !haveUpper {
		return s, nil
	}

	var buf []byte
	for _, r := range s {
		if 'A' <= r && r <= 'Z' {
			buf = append(buf, '!', byte(r+'a'-'A'))
		} else {
			buf = append(buf, byte(r))
		}
	}
	return string(buf), nil
}
