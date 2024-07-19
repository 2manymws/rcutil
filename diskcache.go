package rcutil

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/2manymws/keyrwmutex"
	"github.com/2manymws/rc"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/sync/errgroup"
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

	defaultAdjustPercentage = 80

	reqCacheSuffix = ".request"
	resCacheSuffix = ".response"
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
	cacheRoot            string
	maxKeys              uint64
	maxTotalBytes        uint64
	disableAutoCleanup   bool
	disableWarmUp        bool
	enableAutoAdjust     bool
	adjustTotalBytes     uint64
	enableTouchOnHit     bool
	m                    *ttlcache.Cache[string, *cacheItem]
	d                    *deque
	totalBytes           uint64
	cacheDirLen          int
	mu                   sync.Mutex
	keyMu                *keyrwmutex.KeyRWMutex
	adjustMu             sync.Mutex
	adjustStopCtx        context.Context //nostyle:contexts
	adjustStopCancelFunc context.CancelFunc
	warmUpStopCtx        context.Context //nostyle:contexts
	warmUpStopCancelFunc context.CancelFunc
}

// DiskCacheOption is an option for DiskCache.
type DiskCacheOption func(*DiskCache) error

// MaxKeys sets the maximum number of keys that can be stored in the cache.
func MaxKeys(n uint64) DiskCacheOption {
	return func(c *DiskCache) error {
		c.maxKeys = n
		return nil
	}
}

// MaxTotalBytes sets the maximum number of bytes that can be stored in the cache.
func MaxTotalBytes(n uint64) DiskCacheOption {
	return func(c *DiskCache) error {
		c.maxTotalBytes = n
		return nil
	}
}

// DisableAutoCleanup disables the automatic cache cleanup.
func DisableAutoCleanup() DiskCacheOption {
	return func(c *DiskCache) error {
		c.disableAutoCleanup = true
		return nil
	}
}

// DisableWarmUp disables the automatic cache warm up.
func DisableWarmUp() DiskCacheOption {
	return func(c *DiskCache) error {
		c.disableWarmUp = true
		return nil
	}
}

// EnableAutoAdjust enables auto-adjustment to delete the oldest cache when the total cache size limit (maxTotalBytes) is reached.
func EnableAutoAdjust() DiskCacheOption {
	return func(c *DiskCache) error {
		if c.maxTotalBytes == NoLimitTotalBytes {
			return fmt.Errorf("maxTotalBytes must be set to enable auto-adjust")
		}
		c.enableAutoAdjust = true
		c.adjustTotalBytes = c.maxTotalBytes * defaultAdjustPercentage / 100
		return nil
	}
}

// EnableAutoAdjustWithPercentage enables auto-adjustment to delete the oldest cache when the total cache size limit (maxTotalBytes) is reached.
// percentage: Delete until what percentage of the total byte size is reached.
func EnableAutoAdjustWithPercentage(percentage uint64) DiskCacheOption {
	return func(c *DiskCache) error {
		if c.maxTotalBytes == NoLimitTotalBytes {
			return fmt.Errorf("maxTotalBytes must be set to enable auto-adjust")
		}
		if percentage > 100 {
			return fmt.Errorf("percentage must be less than or equal to 100")
		}
		c.enableAutoAdjust = true
		c.adjustTotalBytes = c.maxTotalBytes * percentage / 100
		return nil
	}
}

// EnableTouchOnHit enables the touch on hit feature.
func EnableTouchOnHit() DiskCacheOption {
	return func(c *DiskCache) error {
		c.enableTouchOnHit = true
		return nil
	}
}

// Metrics returns the metrics of the cache.
type Metrics struct {
	ttlcache.Metrics
	TotalBytes uint64
	KeyCount   uint64
}

type cacheItem struct {
	key     string
	pathkey string
	bytes   uint64
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
	adjustStopCtx, adjustStopCancelFunc := context.WithCancel(context.Background())
	warmUpStopCtx, warmUpStopCancelFunc := context.WithCancel(context.Background())

	c := &DiskCache{
		cacheRoot:            cacheRoot,
		maxKeys:              NoLimitKeys,
		maxTotalBytes:        NoLimitTotalBytes,
		cacheDirLen:          DefaultCacheDirLen,
		keyMu:                keyrwmutex.New(0),
		d:                    newDeque(),
		adjustStopCtx:        adjustStopCtx,
		adjustStopCancelFunc: adjustStopCancelFunc,
		warmUpStopCtx:        warmUpStopCtx,
		warmUpStopCancelFunc: warmUpStopCancelFunc,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
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
		go func() {
			_ = c.warmUpCaches() //nostyle:handlerrors
		}()
	}

	return c, nil
}

// StopAll stops all the goroutines of the cache.
func (c *DiskCache) StopAll() {
	c.StopWarmUp()
	c.StartAutoCleanup()
	c.StopAdjust()
}

// StartAutoCleanup starts the goroutine of automatic cache cleanup
func (c *DiskCache) StartAutoCleanup() {
	go c.m.Start()
}

// StopAutoCleanup stops the auto cleanup cache.
func (c *DiskCache) StopAutoCleanup() {
	c.m.Stop()
}

// StopAdjust
func (c *DiskCache) StopAdjust() {
	c.adjustStopCancelFunc()
}

// StopWarmUp stops the warm up cache.
func (c *DiskCache) StopWarmUp() {
	c.warmUpStopCancelFunc()
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
func (c *DiskCache) StoreWithTTL(key string, req *http.Request, res *http.Response, ttl time.Duration) (err error) {
	c.keyMu.LockKey(key)
	defer func() {
		err = errors.Join(err, c.keyMu.UnlockKey(key))
	}()
	p := filepath.Join(c.cacheRoot, KeyToPath(key, c.cacheDirLen))
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	eg := &errgroup.Group{}
	wb := atomic.Uint64{}
	eg.Go(func() error {
		// Store request
		f, err := os.Create(p + reqCacheSuffix)
		if err != nil {
			return err
		}
		defer f.Close()
		wc := &WriteCounter{Writer: f}
		if err := EncodeReq(req, wc); err != nil {
			return err
		}
		wb.Add(wc.Bytes)
		return nil
	})
	eg.Go(func() error {
		// Store response
		f, err := os.Create(p + resCacheSuffix)
		if err != nil {
			return err
		}
		defer f.Close()
		wc := &WriteCounter{Writer: f}
		if err := EncodeRes(res, wc); err != nil {
			return err
		}
		wb.Add(wc.Bytes)
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	ci := &cacheItem{
		key:     key,
		pathkey: p,
		bytes:   wb.Load(),
	}

	if c.maxTotalBytes != NoLimitTotalBytes {
		c.mu.Lock()
		current := c.totalBytes + wb.Load()
		c.mu.Unlock()
		switch {
		case current < c.maxTotalBytes:
		case c.enableAutoAdjust:
			select {
			case <-c.adjustStopCtx.Done():
				// cache is full
				if err := os.Remove(p + reqCacheSuffix); err != nil {
					return err
				}
				if err := os.Remove(p + resCacheSuffix); err != nil {
					return err
				}
				return fmt.Errorf("%w (%d bytes >= %d bytes)", ErrCacheFull, current, c.maxTotalBytes)
			default:
				go c.removeCachesUntilAdjustTotalBytes()
			}
		default:
			// cache is full
			if err := os.Remove(p + reqCacheSuffix); err != nil {
				return err
			}
			if err := os.Remove(p + resCacheSuffix); err != nil {
				return err
			}
			return fmt.Errorf("%w (%d bytes >= %d bytes)", ErrCacheFull, current, c.maxTotalBytes)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.m.Set(key, ci, ttl)
	c.totalBytes += wb.Load()
	c.d.pushFront(key)
	return nil
}

// Load loads the response from the cache.
func (c *DiskCache) Load(key string) (_ *http.Request, _ *http.Response, err error) {
	c.keyMu.RLockKey(key)
	defer func() {
		err = errors.Join(err, c.keyMu.RUnlockKey(key))
	}()
	i := c.m.Get(key)
	if i == nil {
		return nil, nil, rc.ErrCacheNotFound
	}
	if i.IsExpired() {
		return nil, nil, rc.ErrCacheExpired
	}
	ci := i.Value()

	var (
		req *http.Request
		res *http.Response
	)
	eg := &errgroup.Group{}
	eg.Go(func() error {
		f, err := os.Open(ci.pathkey + reqCacheSuffix)
		if err != nil {
			return err
		}
		// Do not defer f.Close()
		req, err = DecodeReq(f)
		if err != nil {
			return errors.Join(err, f.Close())
		}
		return nil
	})

	eg.Go(func() error {
		f, err := os.Open(ci.pathkey + resCacheSuffix)
		if err != nil {
			return err
		}
		// Do not defer f.Close()
		res, err = DecodeRes(f)
		if err != nil {
			return errors.Join(err, f.Close())
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		c.Delete(key)
		if res != nil {
			err = errors.Join(err, res.Body.Close())
		}
		if req != nil {
			err = errors.Join(err, req.Body.Close())
		}
		return nil, nil, errors.Join(err, rc.ErrCacheNotFound)
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

// warmUpCaches warm up the cache
func (c *DiskCache) warmUpCaches() error {
	return filepath.WalkDir(c.cacheRoot, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, resCacheSuffix) {
			return nil
		}
		// Use response cache to warm up
		rel, err := filepath.Rel(c.cacheRoot, path)
		if err != nil {
			return err
		}
		resi, err := info.Info()
		if err != nil {
			return err
		}
		pathkey := strings.TrimSuffix(path, resCacheSuffix)
		key := PathToKey(strings.TrimSuffix(rel, resCacheSuffix))

		// request cache
		reqpath := pathkey + reqCacheSuffix
		reqi, err := os.Stat(reqpath)
		if err != nil {
			return err
		}
		size := uint64(reqi.Size() + resi.Size())
		c.mu.Lock()
		defer c.mu.Unlock()
		c.totalBytes += size
		_ = c.m.Set(key, &cacheItem{ //nostyle:funcfmt
			key:     key,
			pathkey: pathkey,
			bytes:   size,
		}, ttlcache.DefaultTTL)
		select {
		case <-c.warmUpStopCtx.Done():
			return filepath.SkipAll
		default:
		}
		return nil
	})
}

func (c *DiskCache) removeCachesUntilAdjustTotalBytes() {
	if !c.adjustMu.TryLock() {
		return
	}
	defer c.adjustMu.Unlock()
	for {
		select {
		case <-c.adjustStopCtx.Done():
			return
		default:
		}
		key := c.d.back()
		i := c.m.Get(key)
		if i == nil {
			continue
		}
		ci := i.Value()
		c.removeCache(ci)
		c.Delete(key)
		time.Sleep(1 * time.Millisecond)
		c.mu.Lock()
		if c.totalBytes < c.adjustTotalBytes {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
	}
}

func (c *DiskCache) removeCache(ci *cacheItem) {
	defer func() {
		c.d.remove(ci.key)
	}()
	_ = os.Remove(ci.pathkey + reqCacheSuffix) //nostyle:handlerrors
	_ = os.Remove(ci.pathkey + resCacheSuffix) //nostyle:handlerrors
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
