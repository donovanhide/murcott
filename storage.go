package murcott

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
)

const (
	CreateRosterTableStmt = iota
	LoadRosterStmt
	InsertIdToRosterStmt
	ClearRosterStmt
)

// Storage represents a persistent storage.
type Storage struct {
	db   *sql.DB
	stmt map[int]*sql.Stmt
}

func NewStorage(name string) *Storage {
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil
	}
	s := &Storage{
		db:   db,
		stmt: make(map[int]*sql.Stmt),
	}
	s.init()
	return s
}

func (s *Storage) init() {
	s.prepare(CreateRosterTableStmt, "CREATE TABLE roster (id TEXT)")
	s.exec(CreateRosterTableStmt)

	s.prepare(LoadRosterStmt, "SELECT id FROM roster")
	s.prepare(InsertIdToRosterStmt, "INSERT INTO roster (id) VALUES(?)")
	s.prepare(ClearRosterStmt, "DELETE FROM roster")
}

func (s *Storage) prepare(t int, query string) error {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return err
	}
	s.stmt[t] = stmt
	return nil
}

func (s *Storage) exec(t int, args ...interface{}) (sql.Result, error) {
	if stmt, ok := s.stmt[t]; ok {
		return stmt.Exec(args...)
	} else {
		return nil, errors.New("unregistered stmt")
	}
}

func (s *Storage) query(t int, args ...interface{}) (*sql.Rows, error) {
	if stmt, ok := s.stmt[t]; ok {
		return stmt.Query(args...)
	} else {
		return nil, errors.New("unregistered stmt")
	}
}

func (s *Storage) loadRoster() (*Roster, error) {
	var list []NodeId
	rows, err := s.query(LoadRosterStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		rows.Scan(&id)
		nodeid, err := NewNodeIdFromString(id)
		if err == nil {
			list = append(list, nodeid)
		}
	}
	return &Roster{list: list}, nil
}

func (s *Storage) saveRoster(roster *Roster) error {
	_, err := s.exec(ClearRosterStmt)
	if err != nil {
		return err
	}

	for _, id := range roster.list {
		_, err := s.exec(InsertIdToRosterStmt, id.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) close() {
	for _, stmt := range s.stmt {
		stmt.Close()
	}
	s.db.Close()
}
