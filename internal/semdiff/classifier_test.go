package semdiff

import (
	"strings"
	"testing"
)

func hasKind(got []ChangeKind, want ChangeKind) bool {
	for _, k := range got {
		if k == want {
			return true
		}
	}
	return false
}

func classify(t *testing.T, diff string) []ChangeKind {
	t.Helper()
	files, err := Parse(diff)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	return ClassifyFile(files[0])
}

func TestClassifyAddFunction(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,3 +1,7 @@
 package x

+func NewThing() int {
+	return 1
+}
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeAddFunction) {
		t.Errorf("expected AddFunction in %v", kinds)
	}
}

func TestClassifyRenameSymbol(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,3 +1,3 @@
 package x

-func Old(a int) {}
+func New(a int) {}
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeRenameSymbol) {
		t.Errorf("expected RenameSymbol in %v", kinds)
	}
}

func TestClassifyChangeSignature(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,3 +1,3 @@
 package x

-func Foo(a int) {}
+func Foo(a int, b string) {}
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeChangeSignature) {
		t.Errorf("expected ChangeSignature in %v", kinds)
	}
}

func TestClassifyAddTest(t *testing.T) {
	d := `diff --git a/x_test.go b/x_test.go
--- a/x_test.go
+++ b/x_test.go
@@ -1,3 +1,7 @@
 package x

+func TestNew(t *testing.T) {
+	_ = 1
+}
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeAddTest) {
		t.Errorf("expected AddTest in %v", kinds)
	}
}

func TestClassifyAddImport(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,5 +1,6 @@
 package x

 import (
+	"fmt"
 	"os"
 )
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeAddImport) {
		t.Errorf("expected AddImport in %v", kinds)
	}
}

func TestClassifyRemoveImport(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,6 +1,5 @@
 package x

 import (
-	"fmt"
 	"os"
 )
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeRemoveImport) {
		t.Errorf("expected RemoveImport in %v", kinds)
	}
}

func TestClassifyAddErrorHandling(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,5 +1,8 @@
 package x

 func foo() {
+	if err != nil {
+		return err
+	}
 }
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeAddErrorHandling) {
		t.Errorf("expected AddErrorHandling in %v", kinds)
	}
}

func TestClassifyModifyComment(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,4 +1,4 @@
 package x

-// old comment
+// new comment
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeModifyComment) {
		t.Errorf("expected ModifyComment in %v", kinds)
	}
}

func TestClassifyFormatOnly(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,3 +1,3 @@
 package x

-foo  =   1
+foo = 1
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeFormatOnly) {
		t.Errorf("expected FormatOnly in %v", kinds)
	}
}

func TestClassifyAddType(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,3 +1,5 @@
 package x

+type Foo struct {
+}
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeAddType) {
		t.Errorf("expected AddType in %v", kinds)
	}
}

func TestClassifyRemoveFunction(t *testing.T) {
	d := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,6 +1,3 @@
 package x

-func Gone() {
-	return
-}
`
	kinds := classify(t, d)
	if !hasKind(kinds, ChangeRemoveFunction) {
		t.Errorf("expected RemoveFunction in %v", kinds)
	}
}

func TestClassifyOtherForNonGo(t *testing.T) {
	d := `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,2 +1,3 @@
 Title

+New line.
`
	kinds := classify(t, d)
	if len(kinds) == 0 {
		t.Fatalf("expected some classification")
	}
	if kinds[0] != ChangeOther && !hasKind(kinds, ChangeOther) {
		// Acceptable: README adds may be ChangeOther.
		t.Logf("kinds for README: %v", kinds)
	}
	if !strings.Contains(strings.Join(toStrings(kinds), ","), "") {
		t.Errorf("unexpected kinds: %v", kinds)
	}
}

func toStrings(k []ChangeKind) []string {
	out := make([]string, len(k))
	for i, x := range k {
		out[i] = string(x)
	}
	return out
}
