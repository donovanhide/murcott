package murcott

type Config struct {
	Bootstrap []string
}

var DefaultConfig = Config{
	Bootstrap: []string{
		"localhost:9200-9210",
		"h2so5.net:9200-9210",
	},
}
