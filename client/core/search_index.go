package core

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SearchIndex provides FTS5 keyword search over the live cobra command tree.
// Complements VectorIndex (semantic) with exact/prefix matching.
type SearchIndex struct {
	mu            sync.RWMutex
	db            *sql.DB
	path          string
	removeOnClose bool
}

func NewSearchIndex(dbPath string) (*SearchIndex, error) {
	return newSearchIndex(dbPath, false)
}

func newSearchIndex(dbPath string, removeOnClose bool) (*SearchIndex, error) {
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open search db: %w", err)
	}
	si := &SearchIndex{db: db, path: dbPath, removeOnClose: removeOnClose}
	if _, err := db.Exec(searchIndexSchema); err != nil {
		closeErr := db.Close()
		if removeOnClose {
			closeErr = errors.Join(closeErr, removeSearchIndexFiles(dbPath))
		}
		return nil, errors.Join(fmt.Errorf("create search index schema: %w", err), closeErr)
	}
	return si, nil
}

const searchIndexSchema = `CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
	name, type, category, source, short_desc, long_desc, usage, example, flags,
	ttp UNINDEXED, opsec UNINDEXED, subcommands,
	tokenize='unicode61 remove_diacritics 2'
)`

// Rebuild re-indexes all commands from the given menu sources.
func (si *SearchIndex) Rebuild(sources ...func() []*cobra.Command) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	if si.db == nil {
		return errors.New("search index is closed")
	}

	tx, err := si.db.Begin()
	if err != nil {
		return fmt.Errorf("begin search index rebuild: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DROP TABLE IF EXISTS search_index"); err != nil {
		return fmt.Errorf("drop search index schema: %w", err)
	}
	if _, err := tx.Exec(searchIndexSchema); err != nil {
		return fmt.Errorf("create search index schema: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO search_index(name,type,category,source,short_desc,long_desc,usage,example,flags,ttp,opsec,subcommands) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return fmt.Errorf("prepare search index insert: %w", err)
	}

	for _, src := range sources {
		if src == nil {
			continue
		}
		for _, cmd := range src() {
			if err := indexTree(stmt, cmd); err != nil {
				_ = stmt.Close()
				return err
			}
		}
	}
	if err := stmt.Close(); err != nil {
		return fmt.Errorf("close search index insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit search index rebuild: %w", err)
	}
	return nil
}

func indexTree(stmt *sql.Stmt, cmd *cobra.Command) error {
	if cmd.Hidden {
		return nil
	}
	source := cmd.Annotations["source"]
	if source == "" {
		source = "builtin"
	}
	cmdType := "command"
	if source != "builtin" {
		cmdType = "plugin"
	}

	var flags []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) { flags = append(flags, f.Name+" "+f.Usage) })

	var subs []string
	for _, sub := range cmd.Commands() {
		if !sub.Hidden {
			subs = append(subs, sub.Name())
		}
	}

	if _, err := stmt.Exec(cmd.Name(), cmdType, cmd.GroupID, source, cmd.Short, cmd.Long,
		cmd.UseLine(), cmd.Example, strings.Join(flags, " "),
		cmd.Annotations["ttp"], cmd.Annotations["opsec"], strings.Join(subs, " ")); err != nil {
		return fmt.Errorf("index command %q: %w", cmd.CommandPath(), err)
	}

	for _, sub := range cmd.Commands() {
		if err := indexTree(stmt, sub); err != nil {
			return err
		}
	}
	return nil
}

// SearchResult holds a single FTS5 search hit.
type SearchResult struct {
	Name        string
	Type        string
	Category    string
	Source      string
	Description string
	Usage       string
	TTP         string
	Opsec       string
	Subcommands string
	Snippet     string
	Rank        float64
}

// Search performs FTS5 full-text search.
func (si *SearchIndex) Search(query, typeFilter, category string, limit int) ([]SearchResult, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()
	if si.db == nil {
		return nil, errors.New("search index is closed")
	}

	if limit <= 0 {
		limit = 20
	}
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	var conds []string
	var args []interface{}
	conds = append(conds, "search_index MATCH ?")
	args = append(args, ftsQuery)
	if typeFilter != "" {
		conds = append(conds, "type = ?")
		args = append(args, typeFilter)
	}
	if category != "" {
		conds = append(conds, "category = ?")
		args = append(args, category)
	}
	args = append(args, limit)

	rows, err := si.db.Query(fmt.Sprintf(
		`SELECT name,type,category,source,short_desc,usage,ttp,opsec,subcommands,
		snippet(search_index,4,'**','**','...',15),rank
		FROM search_index WHERE %s ORDER BY rank LIMIT ?`,
		strings.Join(conds, " AND ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		rows.Scan(&r.Name, &r.Type, &r.Category, &r.Source, &r.Description,
			&r.Usage, &r.TTP, &r.Opsec, &r.Subcommands, &r.Snippet, &r.Rank)
		results = append(results, r)
	}
	return results, rows.Err()
}

func (si *SearchIndex) Categories(typeFilter string) ([]string, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()
	if si.db == nil {
		return nil, errors.New("search index is closed")
	}

	query := "SELECT DISTINCT category FROM search_index WHERE category != ''"
	args := []interface{}{}
	if typeFilter != "" {
		query += " AND type = ?"
		args = append(args, typeFilter)
	}
	query += " ORDER BY category"

	rows, err := si.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func buildFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	for _, op := range []string{" AND ", " OR ", " NOT ", "\"", "*"} {
		if strings.Contains(query, op) {
			return query
		}
	}
	words := strings.Fields(query)
	var parts []string
	for _, w := range words {
		w = strings.NewReplacer("\"", "", "(", "", ")", "").Replace(w)
		if w != "" {
			parts = append(parts, "\""+w+"\"")
		}
	}
	if len(parts) == 1 {
		return parts[0] + "*"
	}
	return strings.Join(parts, " AND ")
}

func (si *SearchIndex) Close() error {
	if si == nil {
		return nil
	}

	si.mu.Lock()
	defer si.mu.Unlock()

	if si.db == nil {
		return nil
	}
	db := si.db
	si.db = nil
	closeErr := db.Close()
	if !si.removeOnClose {
		return closeErr
	}
	return errors.Join(closeErr, removeSearchIndexFiles(si.path))
}
