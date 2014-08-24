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

type storage struct {
	db   *sql.DB
	stmt map[int]*sql.Stmt
}

func newStorage(name string) *storage {
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil
	}
	s := &storage{
		db:   db,
		stmt: make(map[int]*sql.Stmt),
	}
	s.init()
	return s
}

func (s *storage) init() {
	s.prepare(CreateRosterTableStmt, "CREATE TABLE roster (id BLOB)")
	s.exec(CreateRosterTableStmt)

	s.prepare(LoadRosterStmt, "SELECT id FROM roster")
	s.prepare(InsertIdToRosterStmt, "INSERT INTO roster (id) VALUES(?)")
	s.prepare(ClearRosterStmt, "DELETE FROM roster")
}

func (s *storage) prepare(t int, query string) error {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return err
	}
	s.stmt[t] = stmt
	return nil
}

func (s *storage) exec(t int, args ...interface{}) (sql.Result, error) {
	if stmt, ok := s.stmt[t]; ok {
		return stmt.Exec(args...)
	} else {
		return nil, errors.New("unregistered stmt")
	}
}

func (s *storage) query(t int, args ...interface{}) (*sql.Rows, error) {
	if stmt, ok := s.stmt[t]; ok {
		return stmt.Query(args...)
	} else {
		return nil, errors.New("unregistered stmt")
	}
}

func (s *storage) loadRoster() ([]NodeId, error) {
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
	return list, nil
}

func (s *storage) saveRoster(roster []NodeId) error {
	_, err := s.exec(ClearRosterStmt)
	if err != nil {
		return err
	}

	for _, id := range roster {
		_, err := s.exec(InsertIdToRosterStmt, id.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *storage) close() {
	for _, stmt := range s.stmt {
		stmt.Close()
	}
	s.db.Close()
}
