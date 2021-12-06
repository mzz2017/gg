package config

type Subscription struct {
	Link          string `toml:"link"`
	Select        string `toml:"select" default:"first"`
	CacheLastNode bool   `toml:"cache_last_node" default:"true"`
}
type Cache struct {
	Subscription CacheSubscription `toml:"subscription"`
}
type CacheSubscription struct {
	LastNode string `toml:"last_node"`
}
type Params struct {
	Node         string       `toml:"node"`
	Subscription Subscription `toml:"subscription"`
	Cache        Cache        `toml:"cache"`
	NoUDP        bool         `toml:"no_udp"`
	TestNode     bool         `toml:"test_node_before_use" default:"true"`
}

var ParamsObj Params
