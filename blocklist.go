package murcott

import (
	"errors"

	"github.com/h2so5/murcott/utils"
)

type BlockList struct {
	list []murcott.NodeID
}

func (r *BlockList) List() []murcott.NodeID {
	return append([]murcott.NodeID(nil), r.list...)
}

func (r *BlockList) Add(id murcott.NodeID) {
	for _, n := range r.list {
		if n.Cmp(id) == 0 {
			return
		}
	}
	r.list = append(r.list, id)
}

func (r *BlockList) Remove(id murcott.NodeID) error {
	for i, n := range r.list {
		if n.Cmp(id) == 0 {
			r.list = append(r.list[:i], r.list[i+1:]...)
			return nil
		}
	}
	return errors.New("item not found")
}

func (r *BlockList) contains(id murcott.NodeID) bool {
	for _, n := range r.list {
		if n.Cmp(id) == 0 {
			return true
		}
	}
	return false
}
