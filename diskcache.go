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
	NoLimitKeys       = 0
	NoLimitTotalBytes = 0
	NoLimitTTL        = ttlcache.NoTTL
)

// DiskCache is a disk cache implementation.
type DiskCache struct {
	cacheRoot     string
	maxTotalBytes int
	m             *ttlcache.Cache[string, *cacheItem]
	totalBytes    int
	mu            sync.Mutex
}

type cacheItem struct {
	path  string
	bytes int
}

// NewDiskCache returns a new DiskCache.
// cacheRoot: the root directory of the cache.
// defaultTTL: the default TTL of the cache.
// maxKeys: the maximum number of keys that can be stored in the cache. If NoLimitKeys is specified, there is no limit.
// maxTotalBytes: the maximum number of bytes that can be stored in the cache. If NoLimitTotalBytes is specified, there is no limit.
func NewDiskCache(cacheRoot string, defaultTTL time.Duration, maxKeys uint64, maxTotalBytes int) *DiskCache {
	opts := []ttlcache.Option[string, *cacheItem]{
		ttlcache.WithTTL[string, *cacheItem](defaultTTL),
	}
	if maxKeys > 0 {
		opts = append(opts, ttlcache.WithCapacity[string, *cacheItem](maxKeys))
	}
	c := &DiskCache{
		cacheRoot:     cacheRoot,
		maxTotalBytes: maxTotalBytes,
		m:             ttlcache.New(opts...),
	}
	c.m.OnEviction(func(ctx context.Context, r ttlcache.EvictionReason, i *ttlcache.Item[string, *cacheItem]) {
		ci := i.Value()
		if err := os.Remove(ci.path); err != nil {
			return
		}
		c.mu.Lock()
		c.totalBytes -= ci.bytes
		c.mu.Unlock()
	})
	return c
}

// Store stores the response in the cache with the default TTL.
func (c *DiskCache) Store(key string, res *http.Response) error {
	return c.StoreWithTTL(key, res, ttlcache.DefaultTTL)
}

// StoreWithTTL stores the response in the cache with the specified TTL.
// If you want to store the response with no TTL, use NoLimitTTL.
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
	if c.maxTotalBytes != NoLimitTotalBytes && c.totalBytes+wc.Bytes >= c.maxTotalBytes {
		if err := os.Remove(p); err != nil {
			return err
		}
		return ErrCacheFull
	}
	c.m.Set(key, ci, ttl)
	c.totalBytes += wc.Bytes
	return nil
}

// Load loads the response from the cache.
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
