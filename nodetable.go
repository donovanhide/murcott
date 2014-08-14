package murcott

type bucket struct {
	Zero  *bucket
	One   *bucket
	Nodes []nodeInfo
}

type nodeTable struct {
	root   *bucket
	selfid NodeId
	k      int
}

func newNodeTable(k int, id NodeId) nodeTable {
	return nodeTable{
		root:   &bucket{},
		selfid: id,
		k:      k,
	}
}

func (p *nodeTable) nearestBucket(id NodeId) *bucket {
	dist := p.selfid.Xor(id)
	b := p.root
	for i := 0; i < dist.BitLen() && b.Zero != nil; i++ {
		if dist.Bit(i) == 0 {
			b = b.Zero
		} else {
			b = b.One
		}
	}
	return b
}

func (p *nodeTable) insert(node nodeInfo) {
	dist := p.selfid.Xor(node.Id)
	b := p.nearestBucket(dist)

	for i, v := range b.Nodes {
		if v.Id.Cmp(node.Id) == 0 {
			b.Nodes = append(append(b.Nodes[:i], b.Nodes[i+1:]...), node)
			return
		}
	}

	if len(b.Nodes) < p.k {
		b.Nodes = append(b.Nodes, node)
	} else {
		b.Nodes[len(b.Nodes)-1] = node
	}
}

func (p *nodeTable) remove(id NodeId) {
	dist := p.selfid.Xor(id)
	b := p.nearestBucket(dist)
	for i, n := range b.Nodes {
		if n.Id.Cmp(id) == 0 {
			b.Nodes = append(b.Nodes[:i], b.Nodes[i+1:]...)
		}
	}
}

func (p *nodeTable) nearestNodes(id NodeId) []nodeInfo {
	dist := p.selfid.Xor(id)
	b := p.nearestBucket(dist)
	return b.Nodes
}

func (p *nodeTable) find(id NodeId) *nodeInfo {
	dist := p.selfid.Xor(id)
	b := p.nearestBucket(dist)
	for _, v := range b.Nodes {
		if v.Id.Cmp(id) == 0 {
			return &v
		}
	}
	return nil
}
