package rcutil

import (
	"container/list"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/2manymws/keyrwmutex"
	"github.com/2manymws/rc"
	"github.com/jellydator/ttlcache/v3"
)

const (
	// NoLimitKeys is a special value that means no limit on the number of keys.
	NoLimitKeys = 0
	// NoLimitTotalBytes is a special value that means no limit on the total number of bytes.
	NoLimitTotalBytes = 0
	// NoLimitTTL is a special value that means no limit on the TTL.
	NoLimitTTL = ttlcache.NoTTL
	// DefaultCacheDirLen is the default length of the cache directory name.
	DefaultCacheDirLen = 2
)

type deque struct {
	mu  sync.Mutex
	m   map[string]*list.Element
	lru *list.List
}

func newDeque() *deque {
	return &deque{
		m:   make(map[string]*list.Element),
		lru: list.New(),
	}
}

func (d *deque) pushFront(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if e, ok := d.m[key]; ok {
		d.lru.MoveToFront(e)
		return
	}
	e := d.lru.PushFront(key)
	d.m[key] = e
}

func (d *deque) back() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lru.Len() == 0 {
		return ""
	}
	return d.lru.Back().Value.(string)
}

func (d *deque) remove(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.m[key]
	if !ok {
		return
	}
	d.lru.Remove(e)
	delete(d.m, key)
}

// DiskCache is a disk cache implementation.
type DiskCache struct {
	cacheRoot          string
	maxKeys            uint64
	maxTotalBytes      uint64
	disableAutoCleanup bool
	disableWarmUp      bool
	enableAutoAdjust   bool
	enableTouchOnHit   bool
	m                  *ttlcache.Cache[string, *cacheItem]
	d                  *deque
	totalBytes         uint64
	cacheDirLen        int
	mu                 sync.Mutex
	keyMu              *keyrwmutex.KeyRWMutex
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

// EnableAutoAdjust enables auto-adjustment to delete the oldest cache when the total cache size limit (maxTotalBytes) is reached.
func EnableAutoAdjust() DiskCacheOption {
	return func(c *DiskCache) {
		c.enableAutoAdjust = true
	}
}

// EnableTouchOnHit enables the touch on hit feature.
func EnableTouchOnHit() DiskCacheOption {
	return func(c *DiskCache) {
		c.enableTouchOnHit = true
	}
}

// Metrics returns the metrics of the cache.
type Metrics struct {
	ttlcache.Metrics
	TotalBytes uint64
	KeyCount   uint64
}

type cacheItem struct {
	key   string
	path  string
	bytes uint64
}

// NewDiskCache returns a new DiskCache.
// cacheRoot: the root directory of the cache.
// defaultTTL: the default TTL of the cache.
// maxKeys: the maximum number of keys that can be stored in the cache. If NoLimitKeys is specified, there is no limit.
// maxTotalBytes: the maximum number of bytes that can be stored in the cache. If NoLimitTotalBytes is specified, there is no limit.
func NewDiskCache(cacheRoot string, defaultTTL time.Duration, opts ...DiskCacheOption) (*DiskCache, error) {
	if ok, err := isWritable(cacheRoot); !ok {
		return nil, fmt.Errorf("cache root %q is not writable: %w", cacheRoot, err)
	}
	c := &DiskCache{
		cacheRoot:     cacheRoot,
		maxKeys:       NoLimitKeys,
		maxTotalBytes: NoLimitTotalBytes,
		cacheDirLen:   DefaultCacheDirLen,
		keyMu:         keyrwmutex.New(0),
		d:             newDeque(),
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
	if !c.enableTouchOnHit {
		mopts = append(mopts, ttlcache.WithDisableTouchOnHit[string, *cacheItem]())
	}
	c.m = ttlcache.New(mopts...)
	c.m.OnEviction(func(ctx context.Context, r ttlcache.EvictionReason, i *ttlcache.Item[string, *cacheItem]) {
		ci := i.Value()
		c.removeCache(ci)
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
			size := uint64(fi.Size())
			c.mu.Lock()
			defer c.mu.Unlock()
			c.totalBytes += size
			c.m.Set(key, &cacheItem{
				path:  path,
				bytes: size,
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
func (c *DiskCache) Store(key string, req *http.Request, res *http.Response) error {
	return c.StoreWithTTL(key, req, res, ttlcache.DefaultTTL)
}

// StoreWithTTL stores the response in the cache with the specified TTL.
// If you want to store the response with no TTL, use NoLimitTTL.
func (c *DiskCache) StoreWithTTL(key string, req *http.Request, res *http.Response, ttl time.Duration) error {
	c.keyMu.LockKey(key)
	defer func() {
		_ = c.keyMu.UnlockKey(key)
	}()
	p := filepath.Join(c.cacheRoot, KeyToPath(key, c.cacheDirLen))
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
	if err := EncodeReqRes(req, res, wc); err != nil {
		return err
	}
	ci := &cacheItem{
		key:   key,
		path:  p,
		bytes: wc.Bytes,
	}

	if c.maxTotalBytes != NoLimitTotalBytes {
		for {
			c.mu.Lock()
			if c.totalBytes+wc.Bytes < c.maxTotalBytes {
				c.mu.Unlock()
				break
			}
			c.mu.Unlock()
			if c.enableAutoAdjust {
				key := c.d.back()
				i := c.m.Get(key)
				ci := i.Value()
				c.removeCache(ci)
				c.Delete(key)
				time.Sleep(1 * time.Millisecond)
				continue
			}
			// cache is full
			if err := os.Remove(p); err != nil {
				return err
			}
			return fmt.Errorf("%w (%d bytes >= %d bytes)", ErrCacheFull, c.totalBytes+wc.Bytes, c.maxTotalBytes)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.m.Set(key, ci, ttl)
	c.totalBytes += wc.Bytes
	c.d.pushFront(key)
	return nil
}

// Load loads the response from the cache.
func (c *DiskCache) Load(key string) (*http.Request, *http.Response, error) {
	c.keyMu.RLockKey(key)
	defer func() {
		_ = c.keyMu.RUnlockKey(key)
	}()
	i := c.m.Get(key)
	if i == nil {
		return nil, nil, rc.ErrCacheNotFound
	}
	if i.IsExpired() {
		return nil, nil, rc.ErrCacheExpired
	}
	ci := i.Value()
	f, err := os.Open(ci.path)
	if err != nil {
		c.Delete(key)
		return nil, nil, rc.ErrCacheNotFound
	}
	defer f.Close()
	req, res, err := DecodeReqRes(f)
	if err != nil {
		return nil, nil, err
	}
	return req, res, nil
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

func (c *DiskCache) removeCache(ci *cacheItem) {
	defer func() {
		c.d.remove(ci.key)
	}()
	_ = os.Remove(ci.path)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.totalBytes < ci.bytes {
		c.totalBytes = 0
	} else {
		c.totalBytes -= ci.bytes
	}
}

func isWritable(dir string) (bool, error) {
	const tmpFile = "tmpfile"
	file, err := os.CreateTemp(dir, tmpFile)
	if err != nil {
		return false, err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	return true, nil
}
