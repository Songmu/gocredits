package gocredits

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestLicenseDirs_set(t *testing.T) {
	ld := &licenseDirs{}
	ld.set(&licenseDir{
		name:    "test",
		version: "v0.1.2",
	})
	ld.set(&licenseDir{
		name:    "test",
		version: "v0.1.3",
	})

	if len(ld.names) != 1 {
		t.Errorf("len(ld.names) should be 1 but: %d", len(ld.names))
	}

	if len(ld.dirs["test"]) != 2 {
		t.Errorf("len(ld.dirs[test]) should be 2 but: %d", len(ld.dirs["test"]))
	}
}

func TestTakeCredits(t *testing.T) {
	setupGoProxy(t)

	tests := []struct {
		name        string
		dir         string
		skipMissing bool
		want        []string
		wantErr     string
	}{
		{
			name: "modules for any platform, excluding test-only and unreachable ones",
			dir:  "multi_platform",
			want: []string{"example.com/common", "example.com/keyring", "example.com/winonly", "example.com/winsvcdep"},
		},
		{
			name: "module replaced by a local directory",
			dir:  "local_replace",
			want: []string{"example.com/local"},
		},
		{
			name: "no dependencies",
			dir:  "gomod_only",
			want: []string{},
		},
		{
			name:    "no Go packages",
			dir:     "no_go",
			wantErr: "no Go packages found in",
		},
		{
			name:    "gocredits can't find the license",
			dir:     "no_license",
			wantErr: `could not find the license for "example.com/nolicense"`,
		},
		{
			name:        "gocredits can't find the license. but skip",
			dir:         "no_license",
			skipMissing: true,
			want:        []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// go list may update go.mod and go.sum, so keep the fixtures intact.
			dir := t.TempDir()
			if err := os.CopyFS(dir, os.DirFS(filepath.Join(testdataDir(t), tt.dir))); err != nil {
				t.Fatal(err)
			}
			licenses, err := takeCredits(dir, tt.skipMissing)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got error %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, l := range licenses[1:] {
				got = append(got, l.Name)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMissedDirs(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":        "module example.com/app\n\ngo 1.21\n",
		"main.go":       "package main\n\nfunc main() {}\n",
		"other/main.go": fmt.Sprintf("//go:build !%s\n\npackage main\n\nfunc main() {}\n", runtime.GOOS),
	}
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The CLI passes a relative path by default.
	t.Chdir(dir)
	r := &depsResolver{dir: ".", pkgs: map[string]*goListPackage{}}
	listed, err := r.list([]string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	var roots []*goListPackage
	for _, p := range listed {
		if !p.DepOnly {
			roots = append(roots, p)
		}
	}
	got, err := r.missedDirs(roots)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"./other"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// setupGoProxy serves the modules under testdata/proxy from a file-based
// module proxy, so that the tests need neither the network nor the user's
// module cache.
func setupGoProxy(t *testing.T) {
	t.Helper()
	proxyDir := t.TempDir()
	srcDir := filepath.Join(testdataDir(t), "proxy")
	entries, err := filepath.Glob(filepath.Join(srcDir, "*", "*@*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, modDir := range entries {
		rel, err := filepath.Rel(srcDir, modDir)
		if err != nil {
			t.Fatal(err)
		}
		modPath, version, _ := strings.Cut(filepath.ToSlash(rel), "@")
		writeProxyModule(t, filepath.Join(proxyDir, filepath.FromSlash(modPath), "@v"), modDir, modPath, version)
	}

	proxyURL := filepath.ToSlash(proxyDir)
	if !strings.HasPrefix(proxyURL, "/") {
		proxyURL = "/" + proxyURL
	}
	t.Setenv("GOPROXY", "file://"+proxyURL)
	t.Setenv("GOPATH", t.TempDir())
	t.Setenv("GOMODCACHE", "")
	t.Setenv("GOFLAGS", "-mod=mod -modcacherw")
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GONOPROXY", "")
	t.Setenv("GOPRIVATE", "")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", "off")
}

func writeProxyModule(t *testing.T, dst, modDir, modPath, version string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	gomod, err := os.ReadFile(filepath.Join(modDir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"list":            version + "\n",
		version + ".info": fmt.Sprintf(`{"Version":%q}`, version),
		version + ".mod":  string(gomod),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dst, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	f, err := os.Create(filepath.Join(dst, version+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	err = filepath.WalkDir(modDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(modDir, path)
		if err != nil {
			return err
		}
		w, err := zw.Create(modPath + "@" + version + "/" + filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		bs, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = w.Write(bs)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(wd, "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
