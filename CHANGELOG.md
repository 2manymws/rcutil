# Changelog

## [v0.16.6](https://github.com/2manymws/rcutil/compare/v0.16.5...v0.16.6) - 2026-06-30

### Fix bug 🐛
- fix: stop recursiveRemoveDir from deleting sibling cache files by @k1LoW in https://github.com/2manymws/rcutil/pull/101
### Other Changes
- chore(deps): bump the dependencies group across 1 directory with 4 updates by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/90
- chore(deps): bump the dependencies group with 3 updates by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/91
- chore(deps): bump the dependencies group with 2 updates by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/92
- chore(deps): bump github.com/docker/cli from 27.4.1+incompatible to 29.2.0+incompatible by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/93
- chore(deps): bump the dependencies group with 2 updates by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/88
- chore(deps): bump github.com/opencontainers/runc from 1.2.8 to 1.3.6 by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/97
- ci(workflows): update Go version configuration to use `oldstable` by @k1LoW in https://github.com/2manymws/rcutil/pull/100
- chore(deps): bump the dependencies group across 1 directory with 5 updates by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/98
- chore(deps): bump the dependencies group across 1 directory with 2 updates by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/99

## [v0.16.5](https://github.com/2manymws/rcutil/compare/v0.16.4...v0.16.5) - 2025-11-06
### Other Changes
- chore: update lint/ci setting by @k1LoW in https://github.com/2manymws/rcutil/pull/80
- chore: update tagpr setting by @k1LoW in https://github.com/2manymws/rcutil/pull/84
- Bump github.com/docker/docker from 26.1.5+incompatible to 28.0.0+incompatible by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/78
- Bump github.com/opencontainers/runc from 1.1.14 to 1.2.8 by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/79
- chore: bump up go directive version by @k1LoW in https://github.com/2manymws/rcutil/pull/85
- chore(deps): bump the dependencies group with 3 updates by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/81
- chore(deps): bump the dependencies group with 5 updates by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/82
- chore(deps): bump github.com/go-viper/mapstructure/v2 from 2.1.0 to 2.4.0 by @dependabot[bot] in https://github.com/2manymws/rcutil/pull/86

## [v0.16.4](https://github.com/2manymws/rcutil/compare/v0.16.3...v0.16.4) - 2024-09-20
### Fix bug 🐛
- When deleting the cache, unnecessary directories are also deleted. by @k1LoW in https://github.com/2manymws/rcutil/pull/75
### Other Changes
- Update go directive version by @k1LoW in https://github.com/2manymws/rcutil/pull/76

## [v0.16.3](https://github.com/2manymws/rcutil/compare/v0.16.2...v0.16.3) - 2024-09-04
### Other Changes
- Use oldstable by @k1LoW in https://github.com/2manymws/rcutil/pull/72
- Bump github.com/opencontainers/runc from 1.1.12 to 1.1.14 by @dependabot in https://github.com/2manymws/rcutil/pull/71
- Bump github.com/docker/docker from 26.1.4+incompatible to 26.1.5+incompatible by @dependabot in https://github.com/2manymws/rcutil/pull/74

## [v0.16.2](https://github.com/2manymws/rcutil/compare/v0.16.1...v0.16.2) - 2024-07-30
### Other Changes
- Update docker packages by @k1LoW in https://github.com/2manymws/rcutil/pull/70

## [v0.16.1](https://github.com/2manymws/rcutil/compare/v0.16.0...v0.16.1) - 2024-07-19
### Other Changes
- Revert "Remove directories recursively when it is zero cache files." by @k1LoW in https://github.com/2manymws/rcutil/pull/66

## [v0.16.0](https://github.com/2manymws/rcutil/compare/v0.15.0...v0.16.0) - 2024-07-12
### New Features 🎉
- Remove directories recursively when it is zero cache files. by @k1LoW in https://github.com/2manymws/rcutil/pull/64

## [v0.15.0](https://github.com/2manymws/rcutil/compare/v0.14.1...v0.15.0) - 2024-07-12
### Breaking Changes 🛠
- Improving DiskCache performance. by @k1LoW in https://github.com/2manymws/rcutil/pull/62
### Other Changes
- Bump github.com/docker/docker from 24.0.7+incompatible to 24.0.9+incompatible by @dependabot in https://github.com/2manymws/rcutil/pull/60
- Add oldstable by @k1LoW in https://github.com/2manymws/rcutil/pull/63

## [v0.14.1](https://github.com/2manymws/rcutil/compare/v0.14.0...v0.14.1) - 2024-03-06
### Fix bug 🐛
- Fix problem of blocking due to ch when StopAll is called. by @k1LoW in https://github.com/2manymws/rcutil/pull/58

## [v0.14.0](https://github.com/2manymws/rcutil/compare/v0.13.0...v0.14.0) - 2024-03-05
### Breaking Changes 🛠
- Asynchronous cache warming by @k1LoW in https://github.com/2manymws/rcutil/pull/56
### Other Changes
- Add StopAdjust and StopAll by @k1LoW in https://github.com/2manymws/rcutil/pull/55

## [v0.13.0](https://github.com/2manymws/rcutil/compare/v0.12.1...v0.13.0) - 2024-03-04
### Breaking Changes 🛠
- Change DiskCacheOption signature by @k1LoW in https://github.com/2manymws/rcutil/pull/51
### Fix bug 🐛
- Fix nil pointer dereference in diskcache.go by @k1LoW in https://github.com/2manymws/rcutil/pull/53
### Other Changes
- Add the option to specify a percentage of the total size to adjust the total size of cache files. by @k1LoW in https://github.com/2manymws/rcutil/pull/54

## [v0.12.1](https://github.com/2manymws/rcutil/compare/v0.12.0...v0.12.1) - 2024-03-01
### Fix bug 🐛
- Fix auto adjust by @k1LoW in https://github.com/2manymws/rcutil/pull/49

## [v0.12.0](https://github.com/2manymws/rcutil/compare/v0.11.1...v0.12.0) - 2024-02-29
### New Features 🎉
- Add EnableAutoAdjust() to enable auto-adjusting the cache size by @k1LoW in https://github.com/2manymws/rcutil/pull/48
### Other Changes
- Bump github.com/opencontainers/runc from 1.1.5 to 1.1.12 by @dependabot in https://github.com/2manymws/rcutil/pull/46

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
