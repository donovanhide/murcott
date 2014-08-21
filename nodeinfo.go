package murcott

import (
	"net"
)

type nodeInfo struct {
	Id   NodeId
	Addr *net.UDPAddr
}

type nodeInfoSorter struct {
	nodes []nodeInfo
	id    NodeId
}

func (p nodeInfoSorter) Len() int {
	return len(p.nodes)
}

func (p nodeInfoSorter) Swap(i, j int) {
	p.nodes[i], p.nodes[j] = p.nodes[j], p.nodes[i]
}

func (p nodeInfoSorter) Less(i, j int) bool {
	disti := p.nodes[i].Id.Xor(p.id)
	distj := p.nodes[j].Id.Xor(p.id)
	return (disti.Cmp(distj) == -1)
}
