package murcott

import (
	"testing"
)

func TestStorageRoster(t *testing.T) {
	s := NewStorage(":memory:")

	roster := []NodeId{
		newRandomNodeId(),
		newRandomNodeId(),
		newRandomNodeId(),
		newRandomNodeId(),
		newRandomNodeId(),
	}

	err := s.saveRoster(roster)
	if err != nil {
		t.Errorf("saveRoster error: %v", err)
	}

	r, err := s.loadRoster()
	if err != nil {
		t.Errorf("loadRoster error: %v", err)
	}

	if len(r) != len(roster) {
		t.Errorf("roster length mismatch")
	}

	for i, id := range r {
		if id.cmp(roster[i]) != 0 {
			t.Errorf("wrong NodeId: %s; expects %s", id.String(), roster[i].String())
		}
	}

	s.close()
}
