package cache

import (
	"documents/internal/storage"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache struct {
	c *cache.Cache
}

func New(liveTime time.Duration) *Cache {
	cache := cache.New(liveTime*time.Minute, liveTime*time.Minute)

	return &Cache{
		c: cache,
	}
}

func (c *Cache) Set(doc storage.Document) {
	c.c.Set(strconv.Itoa(doc.Id), doc, cache.DefaultExpiration)
}

func (c *Cache) Get(id int) (storage.Document, bool) {
	document, found := c.c.Get(strconv.Itoa(id))

	if !found {
		return storage.Document{}, found
	}

	return document.(storage.Document), found
}
