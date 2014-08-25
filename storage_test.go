package murcott

import (
	"testing"
)

func TestStorageRoster(t *testing.T) {
	s := NewStorage(":memory:")

	roster := &Roster{
		list: []NodeId{
			newRandomNodeId(),
			newRandomNodeId(),
			newRandomNodeId(),
			newRandomNodeId(),
			newRandomNodeId(),
		},
	}

	err := s.saveRoster(roster)
	if err != nil {
		t.Errorf("saveRoster error: %v", err)
	}

	r, err := s.loadRoster()
	if err != nil {
		t.Errorf("loadRoster error: %v", err)
	}

	if len(r.list) != len(roster.list) {
		t.Errorf("roster length mismatch")
	}

	for i, id := range r.list {
		if id.cmp(roster.list[i]) != 0 {
			t.Errorf("wrong NodeId: %s; expects %s", id.String(), roster.list[i].String())
		}
	}

	s.close()
}
