package murcott

import (
	"sync"
)

type bucket struct {
	zero  *bucket
	one   *bucket
	nodes []nodeInfo
}

type nodeTable struct {
	root   *bucket
	selfid NodeId
	k      int
	mutex  *sync.Mutex
}

func newNodeTable(k int, id NodeId) nodeTable {
	return nodeTable{
		root:   &bucket{},
		selfid: id,
		k:      k,
		mutex:  &sync.Mutex{},
	}
}

func (p *nodeTable) insert(node nodeInfo) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.selfid.Cmp(node.Id) == 0 {
		return
	}

	b := p.nearestBucket(node.Id)

	for i, v := range b.nodes {
		if v.Id.Cmp(node.Id) == 0 {
			b.nodes = append(append(b.nodes[:i], b.nodes[i+1:]...), node)
			return
		}
	}

	if len(b.nodes) < p.k {
		b.nodes = append(b.nodes, node)
	} else {
		b.nodes[len(b.nodes)-1] = node
	}

	p.devideTree()
}

func (p *nodeTable) remove(id NodeId) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := p.nearestBucket(id)
	for i, n := range b.nodes {
		if n.Id.Cmp(id) == 0 {
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
		}
	}
}

func (p *nodeTable) nearestNodes(id NodeId) []nodeInfo {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := p.nearestBucket(id)
	return append([]nodeInfo(nil), b.nodes...)
}

func (p *nodeTable) find(id NodeId) *nodeInfo {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := p.nearestBucket(id)
	for _, v := range b.nodes {
		if v.Id.Cmp(id) == 0 {
			return &v
		}
	}
	return nil
}

func (p *nodeTable) nearestBucket(id NodeId) *bucket {
	dist := p.selfid.Xor(id)
	b := p.root
	for i := 0; i < dist.BitLen() && b.zero != nil; i++ {
		if dist.Bit(i) == 0 {
			b = b.zero
		} else {
			b = b.one
		}
	}
	return b
}

func (p *nodeTable) devideTree() {
	b := p.root
	i := 0
	for ; b.zero != nil; i++ {
		b = b.zero
	}
	if len(b.nodes) == p.k {
		b.zero = &bucket{}
		b.one = &bucket{}
		for _, n := range b.nodes {
			dist := p.selfid.Xor(n.Id)
			if dist.Bit(i) == 0 {
				b.zero.nodes = append(b.zero.nodes, n)
			} else {
				b.one.nodes = append(b.one.nodes, n)
			}
		}
		b.nodes = b.nodes[:0]
		if len(b.zero.nodes) == p.k {
			p.devideTree()
		}
	}
}
