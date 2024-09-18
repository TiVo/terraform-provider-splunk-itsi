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
	obj  *ItsiObj
	once *sync.Once
}

func (ci *cacheItem) Reset() {
	ci.once = new(sync.Once)
	ci.obj = nil
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

func (c *resourceCache) restkey(obj *ItsiObj) string {
	if obj == nil || (obj.RESTKey == "" && obj.TFID == "") {
		return ""
	}
	restkey := ""
	if obj.RESTKey != "" {
		restkey = obj.RESTKey
	} else if obj.TFID != "" {
		restkey = c.restKey.Get(obj.RestInterface, obj.ObjectType, obj.TFID)
	}
	if restkey == "" {
		return ""
	}
	return formatCacheKey(obj.RestInterface, obj.ObjectType, restkey)
}

func (c *resourceCache) Reset(obj *ItsiObj) *cacheItem {
	k := c.restkey(obj)
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

func (c *resourceCache) Add(obj *ItsiObj) {
	k := c.restkey(obj)
	if k == "" {
		return
	}

	item, ok := c.baseByRESTKey.Get(k)
	if ok {
		item.(*cacheItem).obj = obj
	} else {
		item = &cacheItem{obj: obj}
		c.baseByRESTKey.Add(k, item)
	}
}

func (c *resourceCache) Get(obj *ItsiObj) (item *cacheItem, ok bool) {
	k := c.restkey(obj)
	if k == "" {
		return nil, false
	}
	result, ok := c.baseByRESTKey.Get(k)
	if ok {
		item = result.(*cacheItem)
	}
	return
}

func (c *resourceCache) Remove(obj *ItsiObj) {
	k := c.restkey(obj)
	if k == "" {
		return
	}
	c.baseByRESTKey.Remove(k)
}
