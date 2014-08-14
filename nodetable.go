package murcott

type bucket struct {
	Zero  *bucket
	One   *bucket
	Nodes []nodeInfo
}

type nodeTable struct {
	root *bucket
	k    int
}

func newNodeTable(k int) nodeTable {
	return nodeTable{root: &bucket{}, k: k}
}

func (p *nodeTable) nearestBucket(dist NodeId) *bucket {
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

func (p *nodeTable) insert(node nodeInfo, dist NodeId) {
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

func (p *nodeTable) nearestNodes(dist NodeId) []nodeInfo {
	b := p.nearestBucket(dist)
	return b.Nodes
}

func (p *nodeTable) find(id NodeId, dist NodeId) *nodeInfo {
	b := p.nearestBucket(dist)
	for _, v := range b.Nodes {
		if v.Id.Cmp(id) == 0 {
			return &v
		}
	}
	return nil
}
