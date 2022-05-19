package models

import (
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

var Cache *resourceCache
var cacheMu sync.Mutex

// Cache Item

type cacheItem struct {
	base *Base
	once *sync.Once
}

func (ci *cacheItem) Reset() {
	ci.once = new(sync.Once)
	ci.base = nil
}

// TF ID -> REST Key Mapping

type RESTKeyByTFID struct {
	mapping map[string]string
	mu      sync.Mutex
}

func NewRESTKeyByTFID() *RESTKeyByTFID {
	return &RESTKeyByTFID{mapping: map[string]string{}}
}

func formatCacheKey(RestInterface, ObjectType, key string) string {
	return fmt.Sprintf("%v::%v::%v", RestInterface, ObjectType, key)
}

func (m *RESTKeyByTFID) Get(RestInterface, ObjectType, TFID string) (RESTKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mapping[formatCacheKey(RestInterface, ObjectType, TFID)]
}

func (m *RESTKeyByTFID) Update(RestInterface, ObjectType, TFID string, RESTKey string) {
	if TFID == "" || RESTKey == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mapping[formatCacheKey(RestInterface, ObjectType, TFID)] = RESTKey
}

// Resource Cache

type resourceCache struct {
	baseByRESTKey *lru.Cache
	restKey       *RESTKeyByTFID
}

func NewCache(size int) *resourceCache {
	cacheByRESTKey, err := lru.New(size)
	if err != nil {
		panic(err)
	}
	return &resourceCache{
		baseByRESTKey: cacheByRESTKey,
		restKey:       NewRESTKeyByTFID(),
	}
}

func (c *resourceCache) restkey(b *Base) string {
	if b == nil || (b.RESTKey == "" && b.TFID == "") {
		return ""
	}
	restkey := ""
	if b.RESTKey != "" {
		restkey = b.RESTKey
	} else if b.TFID != "" {
		restkey = c.restKey.Get(b.RestInterface, b.ObjectType, b.TFID)
	}
	if restkey == "" {
		return ""
	}
	return formatCacheKey(b.RestInterface, b.ObjectType, restkey)
}

func (c *resourceCache) Reset(b *Base) *cacheItem {
	k := c.restkey(b)
	if k == "" {
		return &cacheItem{once: new(sync.Once)}
	}

	item, ok := c.baseByRESTKey.Get(k)
	if ok {
		item.(*cacheItem).Reset()
	} else {
		item = &cacheItem{once: new(sync.Once)}
		c.baseByRESTKey.Add(k, item)
	}

	return item.(*cacheItem)
}

func (c *resourceCache) Add(b *Base) {
	k := c.restkey(b)
	if k == "" {
		return
	}

	item, ok := c.baseByRESTKey.Get(k)
	if ok {
		item.(*cacheItem).base = b
	} else {
		item = &cacheItem{base: b}
		c.baseByRESTKey.Add(k, item)
	}
}

func (c *resourceCache) Get(b *Base) (item *cacheItem, ok bool) {
	k := c.restkey(b)
	if k == "" {
		return nil, false
	}
	result, ok := c.baseByRESTKey.Get(k)
	if ok {
		item = result.(*cacheItem)
	}
	return
}

func (c *resourceCache) Remove(b *Base) {
	k := c.restkey(b)
	if k == "" {
		return
	}
	c.baseByRESTKey.Remove(k)
}
