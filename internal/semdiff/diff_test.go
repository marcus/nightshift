package semdiff

import (
	"testing"
)

const sampleDiff = `diff --git a/foo.go b/foo.go
index 1111111..2222222 100644
--- a/foo.go
+++ b/foo.go
@@ -1,5 +1,6 @@
 package foo

-func Old() {}
+func New() {}
+func Helper() {}

diff --git a/bar.go b/bar.go
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/bar.go
@@ -0,0 +1,3 @@
+package bar
+
+func Bar() {}
`

func TestParseMultiFile(t *testing.T) {
	files, err := Parse(sampleDiff)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].NewPath != "foo.go" || files[0].OldPath != "foo.go" {
		t.Errorf("file0 paths: %+v", files[0])
	}
	if !files[1].IsNew {
		t.Errorf("file1 should be flagged as new: %+v", files[1])
	}
	if files[1].NewPath != "bar.go" {
		t.Errorf("file1 NewPath = %q", files[1].NewPath)
	}
	if len(files[0].Hunks) != 1 {
		t.Fatalf("expected 1 hunk in foo.go, got %d", len(files[0].Hunks))
	}
	h := files[0].Hunks[0]
	if len(h.Additions()) != 2 || len(h.Deletions()) != 1 {
		t.Errorf("foo.go hunk lines: adds=%v dels=%v", h.Additions(), h.Deletions())
	}
}

func TestParseHunkHeader(t *testing.T) {
	type want struct{ os, ol, ns, nl int }
	cases := map[string]want{
		"@@ -1,5 +1,6 @@":        {1, 5, 1, 6},
		"@@ -10 +12 @@":          {10, 1, 12, 1},
		"@@ -0,0 +1,3 @@ ctx":    {0, 0, 1, 3},
		"@@ -100,2 +101,2 @@ fn": {100, 2, 101, 2},
	}
	for in, w := range cases {
		got, err := parseHunkHeader(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got.OldStart != w.os || got.OldLines != w.ol || got.NewStart != w.ns || got.NewLines != w.nl {
			t.Errorf("%q: got %+v want %+v", in, got, w)
		}
	}
}

func TestParseRename(t *testing.T) {
	d := `diff --git a/old.go b/new.go
similarity index 90%
rename from old.go
rename to new.go
--- a/old.go
+++ b/new.go
@@ -1,2 +1,2 @@
 package foo
-// old
+// new
`
	files, err := Parse(d)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(files) != 1 || !files[0].IsRename {
		t.Fatalf("rename not detected: %+v", files)
	}
	if files[0].OldPath != "old.go" || files[0].NewPath != "new.go" {
		t.Errorf("rename paths: %+v", files[0])
	}
}

func TestParseEmptyDiff(t *testing.T) {
	files, err := Parse("")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files, got %d", len(files))
	}
}
