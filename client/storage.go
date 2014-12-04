package client

import (
	"database/sql"
	"errors"
	"net"

	"github.com/h2so5/murcott/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vmihailenco/msgpack"
)

const (
	createRosterTableStmt = iota
	loadRosterStmt
	insertIDToRosterStmt
	clearRosterStmt

	createBlockListTableStmt
	loadBlockListStmt
	insertIDToBlockListStmt
	clearBlockListStmt

	createProfileTableStmt
	loadProfileStmt
	insertProfileStmt
	updateProfileStmt

	createKnownNodesTableStmt
	loadKnownNodesStmt
	replaceKnownNodesStmt
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
	s.prepare(insertIDToRosterStmt, "INSERT INTO roster (id) VALUES(?)")
	s.prepare(clearRosterStmt, "DELETE FROM roster")

	s.prepare(createBlockListTableStmt, "CREATE TABLE blocklist (id TEXT)")
	s.exec(createBlockListTableStmt)

	s.prepare(loadBlockListStmt, "SELECT id FROM blocklist")
	s.prepare(insertIDToBlockListStmt, "INSERT INTO blocklist (id) VALUES(?)")
	s.prepare(clearBlockListStmt, "DELETE FROM blocklist")

	s.prepare(createProfileTableStmt, "CREATE TABLE profile (id TEXT, nickname TEXT, data BLOB)")
	s.exec(createProfileTableStmt)

	s.prepare(loadProfileStmt, "SELECT data FROM profile WHERE id = ?")
	s.prepare(insertProfileStmt, "INSERT INTO profile (id, nickname, data) VALUES(?, ?, ?)")
	s.prepare(updateProfileStmt, "UPDATE profile SET nickname = ?, data = ? WHERE id = ?")

	s.prepare(createKnownNodesTableStmt, "CREATE TABLE known_nodes (id TEXT, addr TEXT PRIMARY KEY, updated TIMESTAMP DEFAULT (DATETIME('now','localtime')))")
	s.exec(createKnownNodesTableStmt)

	s.prepare(loadKnownNodesStmt, "SELECT id, addr FROM known_nodes")
	s.prepare(replaceKnownNodesStmt, "REPLACE INTO known_nodes (id, addr) VALUES(?, ?)")
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
	}
	return nil, errors.New("unregistered stmt")
}

func (s *Storage) query(t int, args ...interface{}) (*sql.Rows, error) {
	if stmt, ok := s.stmt[t]; ok {
		return stmt.Query(args...)
	}
	return nil, errors.New("unregistered stmt")
}

func (s *Storage) queryRow(t int, args ...interface{}) *sql.Row {
	if stmt, ok := s.stmt[t]; ok {
		return stmt.QueryRow(args...)
	}
	return nil
}

func (s *Storage) LoadRoster() (*Roster, error) {
	var list []utils.NodeID
	rows, err := s.query(loadRosterStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		rows.Scan(&id)
		nodeid, err := utils.NewNodeIDFromString(id)
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
		_, err := s.exec(insertIDToRosterStmt, id.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) LoadProfile(id utils.NodeID) *UserProfile {
	row := s.queryRow(loadProfileStmt, id.String())
	if row == nil {
		return nil
	}
	var profile UserProfile
	var data []byte
	row.Scan(&data)
	err := msgpack.Unmarshal([]byte(data), &profile)
	if err != nil {
		return nil
	}
	return &profile
}

func (s *Storage) SaveProfile(id utils.NodeID, profile UserProfile) error {
	data, err := msgpack.Marshal(profile)
	if err != nil {
		return err
	}
	if s.LoadProfile(id) == nil {
		_, err := s.exec(insertProfileStmt, id.String(), profile.Nickname, data)
		return err
	}
	_, err = s.exec(updateProfileStmt, profile.Nickname, data, id.String())
	return err
}

func (s *Storage) LoadBlockList() (*BlockList, error) {
	var list []utils.NodeID
	rows, err := s.query(loadBlockListStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		rows.Scan(&id)
		nodeid, err := utils.NewNodeIDFromString(id)
		if err == nil {
			list = append(list, nodeid)
		}
	}
	return &BlockList{list: list}, nil
}

func (s *Storage) SaveBlockList(blocklist *BlockList) error {
	_, err := s.exec(clearBlockListStmt)
	if err != nil {
		return err
	}

	for _, id := range blocklist.list {
		_, err := s.exec(insertIDToBlockListStmt, id.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) LoadKnownNodes() ([]utils.NodeInfo, error) {
	var list []utils.NodeInfo
	rows, err := s.query(loadKnownNodesStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, addrstr string
		rows.Scan(&id, &addrstr)
		nodeid, err := utils.NewNodeIDFromString(id)
		if err == nil {
			addr, err := net.ResolveUDPAddr("udp", addrstr)
			if err == nil {
				list = append(list, utils.NodeInfo{ID: nodeid, Addr: addr})
			}
		}
	}
	return list, nil
}

func (s *Storage) SaveKnownNodes(nodes []utils.NodeInfo) error {
	for _, n := range nodes {
		s.exec(replaceKnownNodesStmt, n.ID.String(), n.Addr.String())
	}
	return nil
}

func (s *Storage) close() {
	for _, stmt := range s.stmt {
		stmt.Close()
	}
	s.db.Close()
}
