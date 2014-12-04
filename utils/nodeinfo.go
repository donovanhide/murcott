package utils

import (
	"net"
	"reflect"

	"github.com/vmihailenco/msgpack"
)

type NodeInfo struct {
	ID   NodeID
	Addr net.Addr
}

type NodeInfoSorter struct {
	Nodes []NodeInfo
	ID    NodeID
}

func (p NodeInfoSorter) Len() int {
	return len(p.Nodes)
}

func (p NodeInfoSorter) Swap(i, j int) {
	p.Nodes[i], p.Nodes[j] = p.Nodes[j], p.Nodes[i]
}

func (p NodeInfoSorter) Less(i, j int) bool {
	disti := p.Nodes[i].ID.Xor(p.ID)
	distj := p.Nodes[j].ID.Xor(p.ID)
	return (disti.Cmp(distj) == -1)
}

func init() {
	msgpack.Register(reflect.TypeOf(NodeInfo{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			info := v.Interface().(NodeInfo)
			return e.Encode(map[string][]byte{
				"id":   info.ID.Bytes(),
				"addr": []byte(info.Addr.String()),
			})
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			i, err := d.DecodeMap()
			if err != nil {
				return err
			}
			m := i.(map[interface{}]interface{})
			if id, ok := m["id"].([]byte); ok {
				if addrstr, ok := m["addr"].([]byte); ok {
					var idbuf [20]byte
					copy(idbuf[:], id)
					addr, err := net.ResolveUDPAddr("udp", string(addrstr))
					if err != nil {
						return err
					}
					v.Set(reflect.ValueOf(NodeInfo{
						ID:   NewNodeID(idbuf),
						Addr: addr,
					}))
				}
			}
			return nil
		})
}
