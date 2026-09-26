package gocredits

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type goListModule struct {
	Path    string
	Version string
	Dir     string
	Main    bool
	Replace *goListModule
}

type goListPackage struct {
	ImportPath     string
	Dir            string
	Standard       bool
	DepOnly        bool
	Module         *goListModule
	GoFiles        []string
	CgoFiles       []string
	IgnoredGoFiles []string
	Error          *struct {
		Err string
	}
}

func (p *goListPackage) excludedByConstraints() bool {
	return len(p.GoFiles) == 0 && len(p.CgoFiles) == 0 && len(p.IgnoredGoFiles) > 0
}

type buildConfig struct {
	GOOS, GOARCH string
	cgo          bool
}

type configSet []uint64

func newConfigSet(n int) configSet {
	return make(configSet, (n+63)/64)
}

func (s configSet) add(i int) {
	s[i/64] |= 1 << (i % 64)
}

func (s configSet) isEmpty() bool {
	for _, w := range s {
		if w != 0 {
			return false
		}
	}
	return true
}

func (s configSet) intersect(o configSet) configSet {
	ret := make(configSet, len(s))
	for i := range s {
		ret[i] = s[i] & o[i]
	}
	return ret
}

func (s configSet) union(o configSet) bool {
	var grown bool
	for i := range s {
		if o[i]&^s[i] != 0 {
			s[i] |= o[i]
			grown = true
		}
	}
	return grown
}

type goFile struct {
	imports []string
	configs configSet
}

type depsResolver struct {
	dir        string
	configs    []buildConfig
	pkgs       map[string]*goListPackage
	files      map[string][]*goFile
	matchCache map[string]configSet
}

// Build constraints are evaluated in-process because running "go list -deps"
// for every GOOS/GOARCH pair would take dozens of invocations. The reachable
// configs are propagated per package rather than checking whether a file
// matches any platform, so that imports never built together, such as a
// Linux-only import of a package reachable only on macOS, are not followed.
func depsFromGoList(dir string) (*licenseDirs, error) {
	configs, err := buildConfigs()
	if err != nil {
		return nil, err
	}
	r := &depsResolver{
		dir:        dir,
		configs:    configs,
		pkgs:       map[string]*goListPackage{},
		files:      map[string][]*goFile{},
		matchCache: map[string]configSet{},
	}
	return r.resolve()
}

func (r *depsResolver) resolve() (*licenseDirs, error) {
	all := newConfigSet(len(r.configs))
	for i := range r.configs {
		all.add(i)
	}

	reach := map[string]configSet{}
	var queue []string
	grow := func(importPath string, s configSet) {
		cur, ok := reach[importPath]
		if !ok {
			cur = newConfigSet(len(r.configs))
			reach[importPath] = cur
		}
		if cur.union(s) {
			queue = append(queue, importPath)
		}
	}

	roots, err := r.listRoots()
	if err != nil {
		return nil, err
	}
	for _, p := range roots {
		grow(p.ImportPath, all)
	}
	if len(queue) == 0 {
		return nil, fmt.Errorf("no Go packages found in %s", r.dir)
	}

	for len(queue) > 0 {
		var (
			unlisted []string
			pending  = map[string]bool{}
		)
		for len(queue) > 0 {
			importPath := queue[0]
			queue = queue[1:]
			p, ok := r.pkgs[importPath]
			if !ok {
				if !pending[importPath] {
					pending[importPath] = true
					unlisted = append(unlisted, importPath)
				}
				continue
			}
			// Packages in the standard library are all covered by the Go license.
			if p.Standard {
				continue
			}
			files, err := r.goFiles(p)
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				s := reach[importPath].intersect(f.configs)
				if s.isEmpty() {
					continue
				}
				for _, imp := range f.imports {
					grow(imp, s)
				}
			}
		}
		if len(unlisted) == 0 {
			break
		}
		if _, err := r.list(unlisted); err != nil {
			return nil, err
		}
		queue = unlisted
	}

	modules := map[string]*licenseDir{}
	for importPath, s := range reach {
		p, ok := r.pkgs[importPath]
		if !ok || s.isEmpty() || p.Standard || p.Module == nil || p.Module.Main {
			continue
		}
		m := p.Module
		l := &licenseDir{name: m.Path, version: m.Version, dir: m.Dir}
		if m.Replace != nil {
			// A replacement by a local directory has no module path of its own to
			// be credited as.
			if m.Replace.Version != "" {
				l.name, l.version = m.Replace.Path, m.Replace.Version
			}
			if l.dir == "" {
				l.dir = m.Replace.Dir
			}
		}
		modules[l.name] = l
	}
	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)

	ld := &licenseDirs{}
	for _, name := range names {
		ld.set(modules[name])
	}
	return ld, nil
}

// "./..." drops the packages whose files are all excluded on the host
// platform, such as a Windows-only command. Only the directories it missed are
// named explicitly, because passing every directory may exceed the command
// line length limit in a large repository.
func (r *depsResolver) listRoots() ([]*goListPackage, error) {
	listed, err := r.list([]string{"./..."})
	if err != nil {
		return nil, err
	}
	var roots []*goListPackage
	for _, p := range listed {
		if !p.DepOnly {
			roots = append(roots, p)
		}
	}
	missed, err := r.missedDirs(roots)
	if err != nil {
		return nil, err
	}
	if len(missed) == 0 {
		return roots, nil
	}
	listed, err = r.list(missed)
	if err != nil {
		return nil, err
	}
	for _, p := range listed {
		if !p.DepOnly {
			roots = append(roots, p)
		}
	}
	return roots, nil
}

func (r *depsResolver) missedDirs(roots []*goListPackage) ([]string, error) {
	matched := map[string]bool{}
	for _, p := range roots {
		matched[realPath(p.Dir)] = true
	}
	dirs, err := packageDirs(r.dir)
	if err != nil {
		return nil, err
	}
	var missed []string
	for _, d := range dirs {
		if !matched[realPath(filepath.Join(r.dir, d))] {
			missed = append(missed, "./"+filepath.ToSlash(d))
		}
	}
	return missed, nil
}

// The skipped directories follow the rules of "./..." in module mode.
func packageDirs(root string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path == root {
				return nil
			}
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
				name == "testdata" || name == "vendor" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") ||
			strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		if len(dirs) == 0 || dirs[len(dirs)-1] != rel {
			dirs = append(dirs, rel)
		}
		return nil
	})
	return dirs, err
}

// go list may report a directory through a different path, e.g. /private/tmp
// for /tmp on macOS.
func realPath(path string) string {
	if p, err := filepath.Abs(path); err == nil {
		path = p
	}
	if p, err := filepath.EvalSymlinks(path); err == nil {
		return p
	}
	return filepath.Clean(path)
}

func buildConfigs() ([]buildConfig, error) {
	out, err := run("go", "tool", "dist", "list", "-json")
	if err != nil {
		return nil, err
	}
	var dists []struct {
		GOOS, GOARCH string
		CgoSupported bool
	}
	if err := json.Unmarshal([]byte(out), &dists); err != nil {
		return nil, err
	}
	var configs []buildConfig
	for _, d := range dists {
		configs = append(configs, buildConfig{GOOS: d.GOOS, GOARCH: d.GOARCH})
		if d.CgoSupported {
			configs = append(configs, buildConfig{GOOS: d.GOOS, GOARCH: d.GOARCH, cgo: true})
		}
	}
	return configs, nil
}

func (r *depsResolver) list(patterns []string) ([]*goListPackage, error) {
	args := append([]string{"list", "-deps", "-e", "-json"}, patterns...)
	out, err := runInDir(r.dir, "go", args...)
	if err != nil {
		return nil, err
	}
	var pkgs []*goListPackage
	dec := json.NewDecoder(strings.NewReader(out))
	for {
		p := &goListPackage{}
		if err := dec.Decode(p); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if p.Error != nil && !p.excludedByConstraints() {
			return nil, fmt.Errorf("failed to list package %q: %s", p.ImportPath, p.Error.Err)
		}
		if _, ok := r.pkgs[p.ImportPath]; !ok {
			r.pkgs[p.ImportPath] = p
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

func (r *depsResolver) goFiles(p *goListPackage) ([]*goFile, error) {
	if files, ok := r.files[p.ImportPath]; ok {
		return files, nil
	}
	var (
		files []*goFile
		fset  = token.NewFileSet()
	)
	for _, names := range [][]string{p.GoFiles, p.CgoFiles, p.IgnoredGoFiles} {
		for _, name := range names {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			src, err := os.ReadFile(filepath.Join(p.Dir, name))
			if err != nil {
				return nil, err
			}
			f, err := parser.ParseFile(fset, name, src, parser.ImportsOnly)
			if err != nil {
				return nil, err
			}
			var (
				imports []string
				isCgo   bool
			)
			for _, spec := range f.Imports {
				imp, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return nil, err
				}
				if imp == "C" {
					isCgo = true
					continue
				}
				imports = append(imports, imp)
			}
			files = append(files, &goFile{
				imports: imports,
				configs: r.matchConfigs(p.Dir, name, src, isCgo),
			})
		}
	}
	r.files[p.ImportPath] = files
	return files, nil
}

func (r *depsResolver) matchConfigs(dir, name string, src []byte, isCgo bool) configSet {
	// Keyed by what MatchFile depends on, since most files share the same one.
	key := fmt.Sprintf("%s\x00%s\x00%t", osArchSuffix(name), constraintLines(src), isCgo)
	if s, ok := r.matchCache[key]; ok {
		return s
	}
	s := newConfigSet(len(r.configs))
	for i, c := range r.configs {
		// go/build leaves files importing "C" to Import rather than MatchFile.
		if isCgo && !c.cgo {
			continue
		}
		ctxt := build.Default
		ctxt.GOOS = c.GOOS
		ctxt.GOARCH = c.GOARCH
		ctxt.CgoEnabled = c.cgo
		// MatchFile reads the file header on every call, so serve it from
		// memory instead of opening the file for each config.
		ctxt.OpenFile = func(string) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(src)), nil
		}
		if ok, _ := ctxt.MatchFile(dir, name); ok {
			s.add(i)
		}
	}
	r.matchCache[key] = s
	return s
}

func osArchSuffix(name string) string {
	name = strings.TrimSuffix(name, ".go")
	i := strings.Index(name, "_")
	if i < 0 {
		return ""
	}
	elems := strings.Split(name[i:], "_")
	if len(elems) > 2 {
		elems = elems[len(elems)-2:]
	}
	return strings.Join(elems, "_")
}

func constraintLines(src []byte) string {
	var lines []string
	scr := bufio.NewScanner(bytes.NewReader(src))
	for scr.Scan() {
		line := strings.TrimSpace(scr.Text())
		if strings.HasPrefix(line, "package ") {
			break
		}
		if constraint.IsGoBuild(line) || constraint.IsPlusBuild(line) {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
