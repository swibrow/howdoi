package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"how do I list files in a directory", []string{"directory", "files", "list"}},
		{"git commit amend", []string{"amend", "commit", "git"}},
		{"", nil},
		{"the a an", nil},
		{"docker docker DOCKER", []string{"docker"}},
		{"find .go files", []string{"files", "find", "go"}},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := extractKeywords(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("extractKeywords(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("extractKeywords(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%q) error: %v", dir, err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestOpenCreatesDB(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer store.Close()

	dbPath := filepath.Join(dir, "memory.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected database file at %s", dbPath)
	}
}

func TestSaveAndSearch(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	err := store.Save(ctx, "list files in directory", "ls -la", "List all files including hidden")
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	results, err := store.Search(ctx, "list files", 10)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Command != "ls -la" {
		t.Errorf("command: got %q, want %q", results[0].Command, "ls -la")
	}
	if results[0].UseCount != 1 {
		t.Errorf("use_count: got %d, want 1", results[0].UseCount)
	}
}

func TestSaveDeduplication(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_ = store.Save(ctx, "list files", "ls -la", "List files")
	_ = store.Save(ctx, "show files in dir", "ls -la", "List files")
	_ = store.Save(ctx, "list directory contents", "ls -la", "List files")

	results, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (deduplicated), got %d", len(results))
	}
	if results[0].UseCount != 3 {
		t.Errorf("use_count: got %d, want 3", results[0].UseCount)
	}
}

func TestSearchRelevanceOrdering(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// "git" only
	_ = store.Save(ctx, "git status", "git status", "Show git status")
	// "git" + "branch" — should rank higher for "git branch" query
	_ = store.Save(ctx, "git branch list", "git branch -a", "List git branches")

	results, err := store.Search(ctx, "git branch", 10)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	// The branch command should be first (matches both keywords)
	if results[0].Command != "git branch -a" {
		t.Errorf("expected 'git branch -a' first, got %q", results[0].Command)
	}
}

func TestSearchNoResults(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_ = store.Save(ctx, "git status", "git status", "Show git status")

	results, err := store.Search(ctx, "kubernetes deploy pods", 10)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchEmptyQuestion(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	results, err := store.Search(ctx, "", 10)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for empty question, got %v", results)
	}
}

func TestList(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_ = store.Save(ctx, "list files", "ls", "List files")
	_ = store.Save(ctx, "git status", "git status", "Show status")
	_ = store.Save(ctx, "docker ps", "docker ps", "List containers")

	results, err := store.List(ctx, 2)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results with limit 2, got %d", len(results))
	}
}

func TestSearchFTSTokenMatching(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// Save an entry with "git" and "files" as keywords
	_ = store.Save(ctx, "git list files", "git ls-files", "List tracked files")
	// Save an entry with "digit" — should NOT match a search for "git"
	_ = store.Save(ctx, "count digit occurrences", "grep -c '[0-9]' file.txt", "Count digits")

	// "files" should match "files" exactly
	results, err := store.Search(ctx, "files", 10)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'files', got %d", len(results))
	}
	if results[0].Command != "git ls-files" {
		t.Errorf("expected 'git ls-files', got %q", results[0].Command)
	}

	// "git" should match "git" but NOT "digit"
	results, err = store.Search(ctx, "git", 10)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'git' (not matching 'digit'), got %d", len(results))
	}
	if results[0].Command != "git ls-files" {
		t.Errorf("expected 'git ls-files', got %q", results[0].Command)
	}
}

func TestClear(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_ = store.Save(ctx, "list files", "ls", "List files")
	_ = store.Save(ctx, "git status", "git status", "Show status")

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear error: %v", err)
	}

	results, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after clear, got %d", len(results))
	}
}
