package testutil

import (
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/sync/errgroup"
)

func WarmUpToCreateCache(t testing.TB, urlstr, hostname string, concurrency, cacherange int) {
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
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("status code is not 200: %d", res.StatusCode)
			}
			return res.Body.Close()
		})
	}
	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}
