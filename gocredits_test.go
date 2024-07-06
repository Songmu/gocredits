package gocredits

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
	tmpd, err := os.MkdirTemp("", "gocredits-")
	if err != nil {
		t.Fatal(err)
	}
	origCache := os.Getenv("GOCACHE")
	os.Setenv("GOCACHE", tmpd)
	defer os.Setenv("GOCACHE", origCache)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmpd); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "install", "github.com/Songmu/gocredits/cmd/gocredits@v0.1.0")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		dir         string
		skipMissing bool
		wantErr     error
	}{
		{"go.sum only", "gosum_only", false, nil},
		{"go.mod only", "gomod_only", false, nil},
		{"there is neither go.mod nor go.sum", "no_gomod_no_gosum", false, fmt.Errorf("use go modules")},
		{"gocredits can't fild the license", "no_license", false, fmt.Errorf("could not find the license for \"github.com/Songmu/no_license_pkg\"")},
		{"gocredits can't fild the license. but skip", "no_license", true, nil},
	}
	for _, tt := range tests {
		dir := filepath.Join(testdataDir(t), tt.dir)
		_, gotErr := takeCredits(dir, tt.skipMissing)
		if !reflect.DeepEqual(gotErr, tt.wantErr) {
			t.Errorf("%s:\ngot  %v\nwant %v", tt.name, gotErr, tt.wantErr)
		}
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
