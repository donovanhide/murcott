package murcott

import (
	"errors"
)

// Roster represents a contact list.
type Roster struct {
	list []NodeId
}

func (r *Roster) List() []NodeId {
	return append([]NodeId(nil), r.list...)
}

func (r *Roster) Add(id NodeId) {
	for _, n := range r.list {
		if n.cmp(id) == 0 {
			return
		}
	}
	r.list = append(r.list, id)
}

func (r *Roster) Remove(id NodeId) error {
	for i, n := range r.list {
		if n.cmp(id) == 0 {
			r.list = append(r.list[:i], r.list[i+1:]...)
			return nil
		}
	}
	return errors.New("item not found")
}
