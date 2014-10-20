package murcott

import (
	"errors"
)

// Roster represents a contact list.
type Roster struct {
	list []NodeID
}

func (r *Roster) List() []NodeID {
	return append([]NodeID(nil), r.list...)
}

func (r *Roster) Add(id NodeID) {
	for _, n := range r.list {
		if n.cmp(id) == 0 {
			return
		}
	}
	r.list = append(r.list, id)
}

func (r *Roster) Remove(id NodeID) error {
	for i, n := range r.list {
		if n.cmp(id) == 0 {
			r.list = append(r.list[:i], r.list[i+1:]...)
			return nil
		}
	}
	return errors.New("item not found")
}
