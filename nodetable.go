package murcott

import "sync"

type nodeTable struct {
	buckets [][]nodeInfo
	selfid  NodeID
	k       int
	mutex   *sync.RWMutex
}

func newNodeTable(k int, id NodeID) nodeTable {
	buckets := make([][]nodeInfo, 160)

	return nodeTable{
		buckets: buckets,
		selfid:  id,
		k:       k,
		mutex:   &sync.RWMutex{},
	}
}

func (p *nodeTable) insert(node nodeInfo) {
	p.remove(node.ID)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	b := node.ID.xor(p.selfid).log2int()

	if len(p.buckets[b]) < p.k {
		p.buckets[b] = append(p.buckets[b], node)
	} else {
		p.buckets[b][len(p.buckets[b])-1] = node
	}
}

func (p *nodeTable) remove(id NodeID) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := id.xor(p.selfid).log2int()
	for i, n := range p.buckets[b] {
		if n.ID.cmp(id) == 0 {
			p.buckets[b] = append(p.buckets[b][:i], p.buckets[b][i+1:]...)
			return
		}
	}
}

func (p *nodeTable) nodes() []nodeInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	var i []nodeInfo
	for _, b := range p.buckets {
		for _, n := range b {
			i = append(i, n)
		}
	}
	return i
}

func (p *nodeTable) nearestNodes(id NodeID) []nodeInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	var n []nodeInfo
	b := id.xor(p.selfid).log2int()
	n = append(n, p.buckets[b]...)
	if len(n) > p.k {
		return n[len(n)-p.k:]
	}
	for i := 0; i < 160; i++ {
		rb := b + i
		if rb < 160 {
			n = append(n, p.buckets[rb]...)
		}
		lb := b - i
		if lb >= 0 {
			n = append(n, p.buckets[lb]...)
		}
	}
	if len(n) > p.k {
		return n[len(n)-p.k:]
	}
	return n
}

func (p *nodeTable) find(id NodeID) *nodeInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	b := id.xor(p.selfid).log2int()
	for _, n := range p.buckets[b] {
		if n.ID.cmp(id) == 0 {
			return &n
		}
	}
	return nil
}
