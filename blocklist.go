package murcott

import (
	"errors"
)

type BlockList struct {
	list []NodeID
}

func (r *BlockList) List() []NodeID {
	return append([]NodeID(nil), r.list...)
}

func (r *BlockList) Add(id NodeID) {
	for _, n := range r.list {
		if n.cmp(id) == 0 {
			return
		}
	}
	r.list = append(r.list, id)
}

func (r *BlockList) Remove(id NodeID) error {
	for i, n := range r.list {
		if n.cmp(id) == 0 {
			r.list = append(r.list[:i], r.list[i+1:]...)
			return nil
		}
	}
	return errors.New("item not found")
}

func (r *BlockList) contains(id NodeID) bool {
	for _, n := range r.list {
		if n.cmp(id) == 0 {
			return true
		}
	}
	return false
}
