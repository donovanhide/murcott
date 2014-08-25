package murcott

import (
	"reflect"
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

func TestStorageProfile(t *testing.T) {
	s := NewStorage(":memory:")
	id := newRandomNodeId()
	profile := UserProfile{
		Nickname: "nick",
		Extension: map[string]string{
			"Location": "Tokyo",
		},
	}

	s.saveProfile(id, profile)

	p := s.loadProfile(id)
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

	s.saveProfile(id, profile)

	p = s.loadProfile(id)
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

	s.close()
}
