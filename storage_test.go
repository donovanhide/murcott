package murcott

import (
	"net"
	"reflect"
	"testing"
)

func TestStorageRoster(t *testing.T) {
	s := NewStorage(":memory:")
	defer s.close()

	roster := &Roster{
		list: []NodeID{
			newRandomNodeID(),
			newRandomNodeID(),
			newRandomNodeID(),
			newRandomNodeID(),
			newRandomNodeID(),
		},
	}

	err := s.SaveRoster(roster)
	if err != nil {
		t.Errorf("saveRoster error: %v", err)
	}

	r, err := s.LoadRoster()
	if err != nil {
		t.Errorf("loadRoster error: %v", err)
	}

	if len(r.list) != len(roster.list) {
		t.Errorf("roster length mismatch")
	}

	for i, id := range r.list {
		if id.cmp(roster.list[i]) != 0 {
			t.Errorf("wrong NodeID: %s; expects %s", id.String(), roster.list[i].String())
		}
	}
}

func TestStorageBlockList(t *testing.T) {
	s := NewStorage(":memory:")
	defer s.close()

	roster := &BlockList{
		list: []NodeID{
			newRandomNodeID(),
			newRandomNodeID(),
			newRandomNodeID(),
			newRandomNodeID(),
			newRandomNodeID(),
		},
	}

	err := s.SaveBlockList(roster)
	if err != nil {
		t.Errorf("saveRoster error: %v", err)
	}

	r, err := s.LoadBlockList()
	if err != nil {
		t.Errorf("loadRoster error: %v", err)
	}

	if len(r.list) != len(roster.list) {
		t.Errorf("roster length mismatch")
	}

	for i, id := range r.list {
		if id.cmp(roster.list[i]) != 0 {
			t.Errorf("wrong NodeID: %s; expects %s", id.String(), roster.list[i].String())
		}
	}
}

func TestStorageProfile(t *testing.T) {
	s := NewStorage(":memory:")
	defer s.close()
	id := newRandomNodeID()
	profile := UserProfile{
		Nickname: "nick",
		Extension: map[string]string{
			"Location": "Tokyo",
		},
	}

	s.SaveProfile(id, profile)

	p := s.LoadProfile(id)
	if p == nil {
		t.Errorf("profile node found")
		return
	}

	if p.Nickname != profile.Nickname {
		t.Errorf("wrong Nickname: %s; expects %s", p.Nickname, profile.Nickname)
	}

	if !reflect.DeepEqual(p.Extension, profile.Extension) {
		t.Errorf("wrong Extension: %v; expects %v", p.Extension, profile.Extension)
	}

	profile.Nickname = "nicknick"
	profile.Extension = map[string]string{
		"Location": "Osaka",
		"Timezone": "UTC+9",
	}

	s.SaveProfile(id, profile)

	p = s.LoadProfile(id)
	if p == nil {
		t.Errorf("profile node found")
		return
	}

	if p.Nickname != profile.Nickname {
		t.Errorf("wrong Nickname: %s; expects %s", p.Nickname, profile.Nickname)
	}

	if !reflect.DeepEqual(p.Extension, profile.Extension) {
		t.Errorf("wrong Extension: %v; expects %v", p.Extension, profile.Extension)
	}
}

func TestStorageKnownNodes(t *testing.T) {
	s := NewStorage(":memory:")
	defer s.close()

	addr1, _ := net.ResolveUDPAddr("udp", "192.168.0.1:2345")
	addr2, _ := net.ResolveUDPAddr("udp", "127.0.0.1:34567")
	addr3, _ := net.ResolveUDPAddr("udp", "19.94.244.34:1234")

	nodes := []nodeInfo{
		nodeInfo{ID: newRandomNodeID(), Addr: addr1},
		nodeInfo{ID: newRandomNodeID(), Addr: addr2},
		nodeInfo{ID: newRandomNodeID(), Addr: addr3},
	}

	err := s.saveKnownNodes(nodes)
	if err != nil {
		t.Errorf("saveKnownNodes failed: %v", err)
	}

	nodes2, err := s.loadKnownNodes()
	if err != nil {
		t.Errorf("loadKnownNodes failed: %v", err)
	}

	if len(nodes2) != len(nodes) {
		t.Errorf("wrong length: %d; expects %d", len(nodes2), len(nodes))
	}

	for i := range nodes2 {
		if nodes2[i].ID.cmp(nodes[i].ID) != 0 {
			t.Errorf("nodeInfo.ID mismatch: %s; expects %s", nodes2[i].ID.String(), nodes[i].ID.String())
		}
		if nodes2[i].Addr.String() != nodes[i].Addr.String() {
			t.Errorf("nodeInfo.Addr mismatch: %s; expects %s", nodes2[i].Addr.String(), nodes[i].Addr.String())
		}
	}
}
