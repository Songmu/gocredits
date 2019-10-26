package gocredits

import "testing"

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
