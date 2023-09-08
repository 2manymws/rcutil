package rcutil

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

const (
	// NoLimitKeys is a special value that means no limit on the number of keys.
	NoLimitKeys = 0
	// NoLimitTotalBytes is a special value that means no limit on the total number of bytes.
	NoLimitTotalBytes = 0
	// NoLimitTTL is a special value that means no limit on the TTL.
	NoLimitTTL = ttlcache.NoTTL
)

// DiskCache is a disk cache implementation.
type DiskCache struct {
	cacheRoot          string
	maxKeys            uint64
	maxTotalBytes      uint64
	disableAutoCleanup bool
	disableWarmUp      bool
	m                  *ttlcache.Cache[string, *cacheItem]
	totalBytes         uint64
	mu                 sync.Mutex
}

// DiskCacheOption is an option for DiskCache.
type DiskCacheOption func(*DiskCache)

// MaxKeys sets the maximum number of keys that can be stored in the cache.
func MaxKeys(n uint64) DiskCacheOption {
	return func(c *DiskCache) {
		c.maxKeys = n
	}
}

// MaxTotalBytes sets the maximum number of bytes that can be stored in the cache.
func MaxTotalBytes(n uint64) DiskCacheOption {
	return func(c *DiskCache) {
		c.maxTotalBytes = n
	}
}

// DisableAutoCleanup disables the automatic cache cleanup.
func DisableAutoCleanup() DiskCacheOption {
	return func(c *DiskCache) {
		c.disableAutoCleanup = true
	}
}

// DisableWarmUp disables the automatic cache warm up.
func DisableWarmUp() DiskCacheOption {
	return func(c *DiskCache) {
		c.disableWarmUp = true
	}
}

// Metrics returns the metrics of the cache.
type Metrics struct {
	ttlcache.Metrics
	TotalBytes uint64
	KeyCount   uint64
}

type cacheItem struct {
	path  string
	bytes uint64
}

// NewDiskCache returns a new DiskCache.
// cacheRoot: the root directory of the cache.
// defaultTTL: the default TTL of the cache.
// maxKeys: the maximum number of keys that can be stored in the cache. If NoLimitKeys is specified, there is no limit.
// maxTotalBytes: the maximum number of bytes that can be stored in the cache. If NoLimitTotalBytes is specified, there is no limit.
func NewDiskCache(cacheRoot string, defaultTTL time.Duration, opts ...DiskCacheOption) (*DiskCache, error) {
	c := &DiskCache{
		cacheRoot:     cacheRoot,
		maxKeys:       NoLimitKeys,
		maxTotalBytes: NoLimitTotalBytes,
	}
	for _, opt := range opts {
		opt(c)
	}

	mopts := []ttlcache.Option[string, *cacheItem]{
		ttlcache.WithTTL[string, *cacheItem](defaultTTL),
	}
	if c.maxKeys > 0 {
		mopts = append(mopts, ttlcache.WithCapacity[string, *cacheItem](c.maxKeys))
	}
	c.m = ttlcache.New(mopts...)
	c.m.OnEviction(func(ctx context.Context, r ttlcache.EvictionReason, i *ttlcache.Item[string, *cacheItem]) {
		ci := i.Value()
		_ = os.Remove(ci.path)
		c.mu.Lock()
		c.totalBytes -= ci.bytes
		c.mu.Unlock()
	})
	if !c.disableAutoCleanup {
		c.StartAutoCleanup()
	}

	if !c.disableWarmUp {
		// Warm up the cache
		if err := filepath.WalkDir(cacheRoot, func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(cacheRoot, path)
			if err != nil {
				return err
			}
			fi, err := info.Info()
			if err != nil {
				return err
			}
			key := strings.ReplaceAll(rel, string(filepath.Separator), "")
			c.m.Set(key, &cacheItem{
				path:  path,
				bytes: uint64(fi.Size()),
			}, ttlcache.DefaultTTL)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// StartAutoCleanup starts the goroutine of automatic cache cleanup
func (c *DiskCache) StartAutoCleanup() {
	go c.m.Start()
}

// StopAutoCleanup stops the auto cleanup cache.
func (c *DiskCache) StopAutoCleanup() {
	c.m.Stop()
}

// DeleteExpired deletes expired caches.
func (c *DiskCache) DeleteExpired() {
	c.m.DeleteExpired()
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
	if err := EncodeResponse(res, wc); err != nil {
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
		c.m.Delete(key)
		return nil, ErrCacheNotFound
	}
	defer f.Close()
	res, err := DecodeResponse(f)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Delete deletes the cache.
func (c *DiskCache) Delete(key string) {
	c.m.Delete(key)
}

// Metrics returns the metrics of the cache.
func (c *DiskCache) Metrics() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := c.m.Metrics()
	return Metrics{
		Metrics:    m,
		TotalBytes: c.totalBytes,
		KeyCount:   uint64(len(c.m.Keys())),
	}
}
