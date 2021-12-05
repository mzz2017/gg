package config

type Subscription struct {
	Link          string `toml:"link"`
	Select        string `toml:"select" default:"first"`
	CacheLastNode bool   `toml:"cachelastnode" default:"true"`
}
type Cache struct {
	Subscription CacheSubscription `toml:"subscription"`
}
type CacheSubscription struct {
	LastNode string `toml:"lastnode"`
}
type Params struct {
	Node         string       `toml:"node"`
	Subscription Subscription `toml:"subscription"`
	Cache        Cache        `toml:"cache"`
	NoUDP        bool         `toml:"noudp"`
	TestNode     bool         `toml:"testnode" default:"true"`
}

var ParamsObj Params
