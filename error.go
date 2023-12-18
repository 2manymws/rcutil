package rcutil

import "errors"

// ErrCacheFull is returned if the cache is full
var ErrCacheFull error = errors.New("cache full")
