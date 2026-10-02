package semdiff

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExplainAggregates(t *testing.T) {
	d := `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,7 @@
 package foo

+func Added() {}
+func AlsoAdded() {}

diff --git a/foo_test.go b/foo_test.go
new file mode 100644
--- /dev/null
+++ b/foo_test.go
@@ -0,0 +1,4 @@
+package foo
+
+func TestAdded(t *testing.T) {}
`
	files, err := Parse(d)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	exp := Explain(files)
	if len(exp.Files) != 2 {
		t.Fatalf("want 2 file explanations, got %d", len(exp.Files))
	}
	if exp.Categories[ChangeAddFunction] < 1 {
		t.Errorf("expected AddFunction count >= 1, got %d", exp.Categories[ChangeAddFunction])
	}
	if exp.Categories[ChangeAddTest] < 1 {
		t.Errorf("expected AddTest count >= 1, got %d", exp.Categories[ChangeAddTest])
	}
	if !strings.Contains(exp.Summary, "2 file(s) changed") {
		t.Errorf("summary missing file count: %q", exp.Summary)
	}
}

func TestExplainRenderJSON(t *testing.T) {
	files, _ := Parse(`diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,2 +1,3 @@
 package x

+func A() {}
`)
	exp := Explain(files)
	out, err := exp.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	var roundtrip Explanation
	if err := json.Unmarshal([]byte(out), &roundtrip); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(roundtrip.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(roundtrip.Files))
	}
}

func TestExplainEmpty(t *testing.T) {
	exp := Explain(nil)
	if exp.Summary != "No changes detected." {
		t.Errorf("unexpected summary: %q", exp.Summary)
	}
}

func TestExplainRenderHumanReadable(t *testing.T) {
	files, _ := Parse(`diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,5 @@
 package foo

+func A() {}
+func B() {}
`)
	out := Explain(files).Render()
	if !strings.Contains(out, "foo.go") {
		t.Errorf("render missing path: %q", out)
	}
	if !strings.Contains(out, "AddFunction") {
		t.Errorf("render missing AddFunction: %q", out)
	}
}
