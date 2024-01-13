package memory

import (
	"sync"
	"time"
)

type item struct {
	data           []byte
	lastActiveTime int64
	expiration     time.Duration
}

var itemPool = &sync.Pool{
	New: func() interface{} {
		return new(item)
	},
}

func acquireItem() *item {
	return itemPool.Get().(*item)
}

func releaseItem(item *item) {
	item.data = item.data[:0]
	item.lastActiveTime = 0
	item.expiration = 0

	itemPool.Put(item)
}
