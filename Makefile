export GO111MODULE=on

default: test

ci: depsdev test

test:
	go test ./... -coverprofile=coverage.out -covermode=count
	cat coverage.out | grep -v "testutil" > coverage.filtered.out
	mv coverage.filtered.out coverage.out
	go test ./... -race

benchmark:
	go test -bench . -benchmem -benchtime 1000x | octocov-go-test-bench --tee > custom_metrics_benchmark.json

lint:
	golangci-lint run ./...
	go vet -vettool=`which gostyle` -gostyle.config=$(PWD)/.gostyle.yml ./...

depsdev:
	go install github.com/Songmu/ghch/cmd/ghch@latest
	go install github.com/Songmu/gocredits/cmd/gocredits@latest
	go install github.com/k1LoW/octocov-go-test-bench/cmd/octocov-go-test-bench@latest
	go install github.com/k1LoW/gostyle@latest

prerelease:
	git pull origin main --tag
	go mod download
	ghch -w -N ${VER}
	gocredits -w .
	git add CHANGELOG.md CREDITS go.mod go.sum
	git commit -m'Bump up version number'
	git tag ${VER}

prerelease_for_tagpr: depsdev
	gocredits -w .
	git add CHANGELOG.md CREDITS go.mod go.sum

release:
	git push origin main --tag
