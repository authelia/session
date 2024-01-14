package redis

var (
	all = []byte(keyWildcard)
)

const (
	keySep         = ":"
	keySepSession  = keySep + "session" + keySep
	keySepLookup   = keySep + "lookup" + keySep
	keyWildcard    = "*"
	keySepWildcard = ":" + keyWildcard
)
