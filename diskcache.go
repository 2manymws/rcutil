package rcutil

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

const (
	NoLimitKeyCapacity   = 0
	NoLimitMaxCacheBytes = 0
)

type DiskCache struct {
	cacheRoot     string
	cacheMaxBytes int
	m             *ttlcache.Cache[string, *cacheItem]
	cacheBytes    int
	mu            sync.Mutex
}

type cacheItem struct {
	path  string
	bytes int
}

func NewDiskCache(cacheRoot string, defaultTTL time.Duration, keyCapacity uint64, cacheMaxBytes int) *DiskCache {
	opts := []ttlcache.Option[string, *cacheItem]{
		ttlcache.WithTTL[string, *cacheItem](defaultTTL),
	}
	if keyCapacity > 0 {
		opts = append(opts, ttlcache.WithCapacity[string, *cacheItem](keyCapacity))
	}
	c := &DiskCache{
		cacheRoot:     cacheRoot,
		cacheMaxBytes: cacheMaxBytes,
		m:             ttlcache.New(opts...),
	}
	c.m.OnEviction(func(ctx context.Context, r ttlcache.EvictionReason, i *ttlcache.Item[string, *cacheItem]) {
		ci := i.Value()
		if err := os.Remove(ci.path); err != nil {
			return
		}
		c.mu.Lock()
		c.cacheBytes -= ci.bytes
		c.mu.Unlock()
	})
	return c
}

func (c *DiskCache) Store(key string, res *http.Response) error {
	return c.StoreWithTTL(key, res, ttlcache.DefaultTTL)
}

func (c *DiskCache) StoreWithTTL(key string, res *http.Response, ttl time.Duration) error {
	p := filepath.Join(c.cacheRoot, KeyToPath(key, 2))
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	wc := &WriteCounter{Writer: f}
	defer f.Close()
	if err := StoreResponse(res, wc); err != nil {
		return err
	}
	ci := &cacheItem{
		path:  p,
		bytes: wc.Bytes,
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cacheMaxBytes != NoLimitMaxCacheBytes && c.cacheBytes+wc.Bytes >= c.cacheMaxBytes {
		if err := os.Remove(p); err != nil {
			return err
		}
		return ErrCacheFull
	}
	c.m.Set(key, ci, ttl)
	c.cacheBytes += wc.Bytes
	return nil
}

func (c *DiskCache) Load(key string) (*http.Response, error) {
	i := c.m.Get(key)
	if i == nil {
		return nil, ErrCacheNotFound
	}
	if i.IsExpired() {
		return nil, ErrCacheExpired
	}
	ci := i.Value()
	f, err := os.Open(ci.path)
	if err != nil {
		return nil, ErrCacheNotFound
	}
	defer f.Close()
	res, err := LoadResponse(f)
	if err != nil {
		return nil, err
	}
	return res, nil
}
