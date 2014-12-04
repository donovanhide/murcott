package client

import (
	"errors"

	"github.com/h2so5/murcott/utils"
)

// Roster represents a contact list.
type Roster struct {
	list []utils.NodeID
}

func (r *Roster) List() []utils.NodeID {
	return append([]utils.NodeID(nil), r.list...)
}

func (r *Roster) Add(id utils.NodeID) {
	for _, n := range r.list {
		if n.Cmp(id) == 0 {
			return
		}
	}
	r.list = append(r.list, id)
}

func (r *Roster) Remove(id utils.NodeID) error {
	for i, n := range r.list {
		if n.Cmp(id) == 0 {
			r.list = append(r.list[:i], r.list[i+1:]...)
			return nil
		}
	}
	return errors.New("item not found")
}
