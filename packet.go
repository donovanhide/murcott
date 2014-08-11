package murcott

type Packet struct {
	Dst     NodeId `msgpack:"dst"`
	Src     NodeId `msgpack:"src"`
	Type    string `msgpack:"type"`
	Payload []byte `msgpack:"payload"`
}
