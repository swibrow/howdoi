package memory

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS interactions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    question    TEXT    NOT NULL,
    command     TEXT    NOT NULL,
    explanation TEXT    NOT NULL DEFAULT '',
    tags        TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    use_count   INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_interactions_created_at ON interactions(created_at);

CREATE VIRTUAL TABLE IF NOT EXISTS interactions_fts USING fts5(
    tags,
    content='interactions',
    content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS interactions_ai AFTER INSERT ON interactions BEGIN
    INSERT INTO interactions_fts(rowid, tags) VALUES (new.id, new.tags);
END;
CREATE TRIGGER IF NOT EXISTS interactions_ad AFTER DELETE ON interactions BEGIN
    INSERT INTO interactions_fts(interactions_fts, rowid, tags) VALUES('delete', old.id, old.tags);
END;
CREATE TRIGGER IF NOT EXISTS interactions_au AFTER UPDATE ON interactions BEGIN
    INSERT INTO interactions_fts(interactions_fts, rowid, tags) VALUES('delete', old.id, old.tags);
    INSERT INTO interactions_fts(rowid, tags) VALUES (new.id, new.tags);
END;
`

type Interaction struct {
	ID          int64
	Question    string
	Command     string
	Explanation string
	Tags        string
	CreatedAt   time.Time
	UseCount    int
}

type Store struct {
	db *sql.DB
}

func Open(dir string) (*Store, error) {
	dbPath := filepath.Join(dir, "memory.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	// Rebuild FTS index to handle existing data from before the FTS migration
	if _, err := db.Exec("INSERT INTO interactions_fts(interactions_fts) VALUES('rebuild')"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("rebuilding FTS index: %w", err)
	}

	// Drop legacy index if it exists (FTS5 replaces it)
	_, _ = db.Exec("DROP INDEX IF EXISTS idx_interactions_tags")

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Save(ctx context.Context, question, command, explanation string) error {
	tags := strings.Join(extractKeywords(question), " ")

	// Upsert: if the same command exists, increment use_count and update question/tags
	result, err := s.db.ExecContext(ctx,
		`UPDATE interactions SET use_count = use_count + 1, question = ?, tags = ?, explanation = ?
		 WHERE command = ?`,
		question, tags, explanation, command,
	)
	if err != nil {
		return fmt.Errorf("updating interaction: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}

	if rows > 0 {
		return nil
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO interactions (question, command, explanation, tags) VALUES (?, ?, ?, ?)`,
		question, command, explanation, tags,
	)
	if err != nil {
		return fmt.Errorf("inserting interaction: %w", err)
	}

	return nil
}

func (s *Store) Search(ctx context.Context, question string, limit int) ([]Interaction, error) {
	keywords := extractKeywords(question)
	if len(keywords) == 0 {
		return nil, nil
	}

	// FTS5 query: join keywords with OR for broad matching
	ftsQuery := strings.Join(keywords, " OR ")

	query := `SELECT i.id, i.question, i.command, i.explanation, i.tags, i.created_at, i.use_count
		 FROM interactions_fts
		 JOIN interactions i ON i.id = interactions_fts.rowid
		 WHERE interactions_fts.tags MATCH ?
		 ORDER BY bm25(interactions_fts) ASC, i.use_count DESC, i.created_at DESC
		 LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("searching interactions: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	return scanInteractions(rows)
}

func (s *Store) List(ctx context.Context, limit int) ([]Interaction, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, question, command, explanation, tags, created_at, use_count
		 FROM interactions
		 ORDER BY created_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing interactions: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	return scanInteractions(rows)
}

func (s *Store) Clear(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM interactions")
	if err != nil {
		return fmt.Errorf("clearing interactions: %w", err)
	}
	return nil
}

func scanInteractions(rows *sql.Rows) ([]Interaction, error) {
	var interactions []Interaction
	for rows.Next() {
		var ix Interaction
		var createdAt string
		if err := rows.Scan(&ix.ID, &ix.Question, &ix.Command, &ix.Explanation, &ix.Tags, &createdAt, &ix.UseCount); err != nil {
			return nil, fmt.Errorf("scanning interaction: %w", err)
		}
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			t, _ = time.Parse("2006-01-02T15:04:05Z", createdAt)
		}
		ix.CreatedAt = t
		interactions = append(interactions, ix)
	}
	return interactions, rows.Err()
}
