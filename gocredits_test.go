package gocredits

import (
	"fmt"
	"os"
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
	tests := []struct {
		name          string
		dir           string
		skipNoLicense bool
		wantErr       error
	}{
		{"go.sub only", "gosum_only", false, nil},
		{"go.mod only", "gomod_only", false, nil},
		{"there is neither go.mod nor go.sum", "no_gomod_no_gosum", false, fmt.Errorf("use go modules")},
		{"no licenses package found", "no_license", false, fmt.Errorf("no licenses found for \"github.com/Songmu/no_license_pkg\"")},
		{"no licenses package found. but skip", "no_license", true, nil},
	}
	for _, tt := range tests {
		dir := filepath.Join(testdataDir(), tt.dir)
		_, gotErr := takeCredits(dir, tt.skipNoLicense)
		if !reflect.DeepEqual(gotErr, tt.wantErr) {
			t.Errorf("%s:\ngot  %v\nwant %v", tt.name, gotErr, tt.wantErr)
		}
	}
}

func testdataDir() string {
	wd, _ := os.Getwd()
	dir, _ := filepath.Abs(filepath.Join(wd, "testdata"))
	return dir
}
