package memory

import (
	"sync"
	"time"
)

type databaseItem struct {
	data           []byte
	lookup         string
	lastActiveTime int64
	expiration     time.Duration
}

var itemPool = &sync.Pool{
	New: func() interface{} {
		return new(databaseItem)
	},
}

func acquireDatabaseItem() *databaseItem {
	return itemPool.Get().(*databaseItem)
}

func releaseDatabaseItem(item *databaseItem) {
	item.data = item.data[:0]
	item.lastActiveTime = 0
	item.expiration = 0

	itemPool.Put(item)
}
