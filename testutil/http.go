package testutil

import (
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/sync/errgroup"
)

func WarmUpToCreateCache(t testing.TB, urlstr, hostname string, concurrency, cacherange) {
	limitCh := make(chan struct{}, concurrency)
	eg := new(errgroup.Group)
	for i := 0; i < cacherange; i++ {
		i := i
		limitCh <- struct{}{}
		eg.Go(func() error {
			defer func() {
				<-limitCh
			}()
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/cache/%d", urlstr, i), nil)
			if err != nil {
				return err
			}
			req.Host = hostname
			req.Close = true
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			return res.Body.Close()
		})
	}
	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}
