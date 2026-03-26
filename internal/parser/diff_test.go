package parser

import (
	"testing"
)

func TestParseUnifiedDiff(t *testing.T) {
	diff := `diff --git a/charts/cert-manager/Chart.yaml b/charts/cert-manager/Chart.yaml
index abc1234..def5678 100644
--- a/charts/cert-manager/Chart.yaml
+++ b/charts/cert-manager/Chart.yaml
@@ -1,5 +1,5 @@
 name: cert-manager
-version: 1.14.5
+version: 1.14.7
 description: A Helm chart for cert-manager
diff --git a/apps/argocd/values.yaml b/apps/argocd/values.yaml
index 111aaa..222bbb 100644
--- a/apps/argocd/values.yaml
+++ b/apps/argocd/values.yaml
@@ -3,7 +3,7 @@
 image:
   repository: quay.io/argoproj/argocd
-  tag: v2.10.6
+  tag: v2.11.0
`

	diffs := ParseUnifiedDiff(diff)

	if len(diffs) != 2 {
		t.Fatalf("expected 2 file diffs, got %d", len(diffs))
	}

	// First file
	if diffs[0].Path != "charts/cert-manager/Chart.yaml" {
		t.Errorf("expected path 'charts/cert-manager/Chart.yaml', got %q", diffs[0].Path)
	}
	if len(diffs[0].RemovedLines) != 1 {
		t.Errorf("expected 1 removed line, got %d", len(diffs[0].RemovedLines))
	}
	if len(diffs[0].AddedLines) != 1 {
		t.Errorf("expected 1 added line, got %d", len(diffs[0].AddedLines))
	}

	// Second file
	if diffs[1].Path != "apps/argocd/values.yaml" {
		t.Errorf("expected path 'apps/argocd/values.yaml', got %q", diffs[1].Path)
	}
	if len(diffs[1].RemovedLines) != 1 {
		t.Errorf("expected 1 removed line, got %d", len(diffs[1].RemovedLines))
	}
	if len(diffs[1].AddedLines) != 1 {
		t.Errorf("expected 1 added line, got %d", len(diffs[1].AddedLines))
	}
}

func TestParseUnifiedDiff_Empty(t *testing.T) {
	diffs := ParseUnifiedDiff("")
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs for empty input, got %d", len(diffs))
	}
}

func TestChangedFiles(t *testing.T) {
	diffs := []FileDiff{
		{Path: "a.yaml"},
		{Path: "b/c.yaml"},
	}
	files := ChangedFiles(diffs)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0] != "a.yaml" || files[1] != "b/c.yaml" {
		t.Errorf("unexpected files: %v", files)
	}
}
