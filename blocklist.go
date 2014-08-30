package murcott

import (
	"errors"
)

type BlockList struct {
	list []NodeId
}

func (r *BlockList) List() []NodeId {
	return append([]NodeId(nil), r.list...)
}

func (r *BlockList) Add(id NodeId) {
	for _, n := range r.list {
		if n.cmp(id) == 0 {
			return
		}
	}
	r.list = append(r.list, id)
}

func (r *BlockList) Remove(id NodeId) error {
	for i, n := range r.list {
		if n.cmp(id) == 0 {
			r.list = append(r.list[:i], r.list[i+1:]...)
			return nil
		}
	}
	return errors.New("item not found")
}

func (r *BlockList) contains(id NodeId) bool {
	for _, n := range r.list {
		if n.cmp(id) == 0 {
			return true
		}
	}
	return false
}
