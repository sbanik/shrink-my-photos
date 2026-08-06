package processor

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDuplicateReviewDeletesOnlyApprovedCandidates(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	keep := filepath.Join(first, "keep.png")
	remove := filepath.Join(second, "remove.png")
	unique := filepath.Join(second, "unique.png")
	for path, content := range map[string]string{keep: "same", remove: "same", unique: "different"} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	deleted := RunDuplicateReview([]string{first, second}, []string{".png"}, bufio.NewReader(strings.NewReader("yes\n")))
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("first selected copy should be kept")
	}
	if _, err := os.Stat(remove); !os.IsNotExist(err) {
		t.Fatal("approved duplicate should be deleted")
	}
	if _, err := os.Stat(unique); err != nil {
		t.Fatal("unique image should not be deleted")
	}
}

func TestRunDuplicateReviewKeepsCandidatesByDefault(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	keep := filepath.Join(first, "keep.jpg")
	candidate := filepath.Join(second, "candidate.jpg")
	for _, path := range []string{keep, candidate} {
		if err := os.WriteFile(path, []byte("same"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if deleted := RunDuplicateReview([]string{first, second}, []string{".jpg"}, bufio.NewReader(strings.NewReader("no\n"))); deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
	if _, err := os.Stat(candidate); err != nil {
		t.Fatal("unapproved duplicate should remain")
	}
}

func TestRunDuplicateReviewDoesNotCompareAFileWithItselfForNestedFolders(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	image := filepath.Join(child, "only.png")
	if err := os.WriteFile(image, []byte("only copy"), 0644); err != nil {
		t.Fatal(err)
	}
	if deleted := RunDuplicateReview([]string{root, child}, []string{".png"}, bufio.NewReader(strings.NewReader("yes\n"))); deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
	if _, err := os.Stat(image); err != nil {
		t.Fatal("a file discovered through two selected roots must remain")
	}
}
