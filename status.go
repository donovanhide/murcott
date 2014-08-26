package murcott

const (
	StatusOffline = "offline"
	StatusAway    = "away"
	StatusActive  = "active"
)

type UserStatus struct {
	Type    string `msgpack:"type"`
	Message string `msgpack:"message"`
}
