package murcott

type bucket struct {
	Zero  *bucket
	One   *bucket
	Nodes []NodeInfo
}

type NodeTable struct {
	root *bucket
	k    int
}

func NewNodeTable(k int) NodeTable {
	return NodeTable{root: &bucket{}, k: k}
}

func (p *NodeTable) nearestBucket(dist NodeId) *bucket {
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

func (p *NodeTable) Insert(node NodeInfo, dist NodeId) {
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

func (p *NodeTable) NearestNodes(dist NodeId) []NodeInfo {
	b := p.nearestBucket(dist)
	return b.Nodes
}

func (p *NodeTable) Find(id NodeId, dist NodeId) *NodeInfo {
	b := p.nearestBucket(dist)
	for _, v := range b.Nodes {
		if v.Id.Cmp(id) == 0 {
			return &v
		}
	}
	return nil
}
