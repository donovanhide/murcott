package murcott

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vmihailenco/msgpack"
)

const (
	createRosterTableStmt = iota
	loadRosterStmt
	insertIdToRosterStmt
	clearRosterStmt

	createProfileTableStmt
	loadProfileStmt
	insertProfileStmt
	updateProfileStmt
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
	s.prepare(createRosterTableStmt, "CREATE TABLE roster (id TEXT)")
	s.exec(createRosterTableStmt)

	s.prepare(loadRosterStmt, "SELECT id FROM roster")
	s.prepare(insertIdToRosterStmt, "INSERT INTO roster (id) VALUES(?)")
	s.prepare(clearRosterStmt, "DELETE FROM roster")

	s.prepare(createProfileTableStmt, "CREATE TABLE profile (id TEXT, nickname TEXT, data BLOB)")
	s.exec(createProfileTableStmt)

	s.prepare(loadProfileStmt, "SELECT data FROM profile WHERE id = ?")
	s.prepare(insertProfileStmt, "INSERT INTO profile (id, nickname, data) VALUES(?, ?, ?)")
	s.prepare(updateProfileStmt, "UPDATE profile SET nickname = ?, data = ? WHERE id = ?")
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

func (s *Storage) queryRow(t int, args ...interface{}) *sql.Row {
	if stmt, ok := s.stmt[t]; ok {
		return stmt.QueryRow(args...)
	} else {
		return nil
	}
}

func (s *Storage) LoadRoster() (*Roster, error) {
	var list []NodeId
	rows, err := s.query(loadRosterStmt)
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

func (s *Storage) SaveRoster(roster *Roster) error {
	_, err := s.exec(clearRosterStmt)
	if err != nil {
		return err
	}

	for _, id := range roster.list {
		_, err := s.exec(insertIdToRosterStmt, id.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) LoadProfile(id NodeId) *UserProfile {
	row := s.queryRow(loadProfileStmt, id.String())
	if row == nil {
		return nil
	} else {
		var profile UserProfile
		var data string
		row.Scan(&data)
		err := msgpack.Unmarshal([]byte(data), &profile)
		if err != nil {
			return nil
		} else {
			return &profile
		}
	}
}

func (s *Storage) SaveProfile(id NodeId, profile UserProfile) error {
	data, err := msgpack.Marshal(profile)
	if err != nil {
		return err
	}
	if s.LoadProfile(id) == nil {
		_, err := s.exec(insertProfileStmt, id.String(), profile.Nickname, string(data))
		return err
	} else {
		_, err := s.exec(updateProfileStmt, profile.Nickname, string(data), id.String())
		return err
	}
}

func (s *Storage) close() {
	for _, stmt := range s.stmt {
		stmt.Close()
	}
	s.db.Close()
}
