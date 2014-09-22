package murcott

import (
	"net"
	"reflect"

	"github.com/vmihailenco/msgpack"
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
	disti := p.nodes[i].Id.xor(p.id)
	distj := p.nodes[j].Id.xor(p.id)
	return (disti.cmp(distj) == -1)
}

func init() {
	msgpack.Register(reflect.TypeOf(nodeInfo{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			info := v.Interface().(nodeInfo)
			return e.Encode(map[string]string{
				"id":   string(info.Id.Bytes()),
				"addr": info.Addr.String(),
			})
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			i, err := d.DecodeMap()
			if err != nil {
				return nil
			}
			m := i.(map[interface{}]interface{})
			if id, ok := m["id"].(string); ok {
				if addrstr, ok := m["addr"].(string); ok {
					var idbuf [20]byte
					copy(idbuf[:], []byte(id))
					addr, err := net.ResolveUDPAddr("udp", addrstr)
					if err != nil {
						return nil
					}
					v.Set(reflect.ValueOf(nodeInfo{
						Id:   NewNodeId(idbuf),
						Addr: addr,
					}))
				}
			}
			return nil
		})
}
