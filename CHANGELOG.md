# Changelog

## [v0.11.1](https://github.com/2manymws/rcutil/compare/v0.11.0...v0.11.1) - 2024-01-30
### Fix bug 🐛
- Fix total bytes negative overflow by @k1LoW in https://github.com/2manymws/rcutil/pull/44

## [v0.11.0](https://github.com/2manymws/rcutil/compare/v0.10.0...v0.11.0) - 2024-01-18
### Breaking Changes 🛠
- Use gob instead of json for cache by @k1LoW in https://github.com/2manymws/rcutil/pull/41
### Other Changes
- Add benchmark for encoding/decoding by @k1LoW in https://github.com/2manymws/rcutil/pull/39
- Fix benchmark by @k1LoW in https://github.com/2manymws/rcutil/pull/42
- Add test for encoding images in the cache. by @k1LoW in https://github.com/2manymws/rcutil/pull/43

## [v0.10.0](https://github.com/2manymws/rcutil/compare/v0.9.0...v0.10.0) - 2024-01-17
### New Features 🎉
- Use RWLock when Load/Store caches by @k1LoW in https://github.com/2manymws/rcutil/pull/38
### Other Changes
- Use rc v0.9.0 by @k1LoW in https://github.com/2manymws/rcutil/pull/35
- Use rc v0.9.1 by @k1LoW in https://github.com/2manymws/rcutil/pull/36

## [v0.9.0](https://github.com/2manymws/rcutil/compare/v0.8.2...v0.9.0) - 2024-01-04
### Breaking Changes 🛠
- Use req.Host ( does not use req.URL.Host ) by @k1LoW in https://github.com/2manymws/rcutil/pull/33

## [v0.8.2](https://github.com/2manymws/rcutil/compare/v0.8.1...v0.8.2) - 2023-12-23
### Other Changes
- Set error for Seed() by @k1LoW in https://github.com/2manymws/rcutil/pull/31

## [v0.8.1](https://github.com/2manymws/rcutil/compare/v0.8.0...v0.8.1) - 2023-12-22
### Breaking Changes 🛠
- Fix Seed logic by @k1LoW in https://github.com/2manymws/rcutil/pull/29

## [v0.8.0](https://github.com/2manymws/rcutil/compare/v0.7.3...v0.8.0) - 2023-12-22
### New Features 🎉
- Support for changing the directory name length in the disk cache. by @k1LoW in https://github.com/2manymws/rcutil/pull/27

## [v0.7.3](https://github.com/2manymws/rcutil/compare/v0.7.2...v0.7.3) - 2023-12-22
### Other Changes
- Check cacheRoot writable by @k1LoW in https://github.com/2manymws/rcutil/pull/25

## [v0.7.2](https://github.com/2manymws/rcutil/compare/v0.7.1...v0.7.2) - 2023-12-18
### Other Changes
- Usr rc.Err* by @k1LoW in https://github.com/2manymws/rcutil/pull/24

## [v0.7.1](https://github.com/2manymws/rcutil/compare/v0.7.0...v0.7.1) - 2023-12-15

## [v0.7.0](https://github.com/k1LoW/rcutil/compare/v0.6.0...v0.7.0) - 2023-12-15
### Breaking Changes 🛠
- Disable touch on hit by default by @k1LoW in https://github.com/k1LoW/rcutil/pull/19
### Fix bug 🐛
- Fix Benchmark by @k1LoW in https://github.com/k1LoW/rcutil/pull/21

## [v0.6.0](https://github.com/k1LoW/rcutil/compare/v0.5.0...v0.6.0) - 2023-12-14
### Breaking Changes 🛠
- Support for rc v0.4.0 by @k1LoW in https://github.com/k1LoW/rcutil/pull/18
### Other Changes
- Add gostyle-action by @k1LoW in https://github.com/k1LoW/rcutil/pull/15
- Bump github.com/docker/docker from 20.10.24+incompatible to 24.0.7+incompatible by @dependabot in https://github.com/k1LoW/rcutil/pull/17

## [v0.5.0](https://github.com/k1LoW/rcutil/compare/v0.4.0...v0.5.0) - 2023-09-08
### Breaking Changes 🛠
- Support warm-up of cache by @k1LoW in https://github.com/k1LoW/rcutil/pull/13

## [v0.4.0](https://github.com/k1LoW/rcutil/compare/v0.3.1...v0.4.0) - 2023-09-08
### Breaking Changes 🛠
- Fix DiskCache sig by @k1LoW in https://github.com/k1LoW/rcutil/pull/11

## [v0.3.1](https://github.com/k1LoW/rcutil/compare/v0.3.0...v0.3.1) - 2023-09-08
### New Features 🎉
- Add SetCacheResultHeader by @k1LoW in https://github.com/k1LoW/rcutil/pull/9

## [v0.3.0](https://github.com/k1LoW/rcutil/compare/v0.2.0...v0.3.0) - 2023-09-08
### Breaking Changes 🛠
- Change funcsions ( use io.Reader / io.Writer instead of []byte ) by @k1LoW in https://github.com/k1LoW/rcutil/pull/7
- Add DiskCache using ttlcache by @k1LoW in https://github.com/k1LoW/rcutil/pull/8

## [v0.2.0](https://github.com/k1LoW/rcutil/compare/v0.1.0...v0.2.0) - 2023-09-07
### New Features 🎉
- Add Seed for generate seed for cache key by @k1LoW in https://github.com/k1LoW/rcutil/pull/5
### Other Changes
- Add benchmmark by @k1LoW in https://github.com/k1LoW/rcutil/pull/2
- Bump github.com/docker/docker from 20.10.7+incompatible to 20.10.24+incompatible by @dependabot in https://github.com/k1LoW/rcutil/pull/4

## [v0.0.1](https://github.com/k1LoW/rcutil/commits/v0.0.1) - 2023-09-05
