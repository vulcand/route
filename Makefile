.PHONY: prof

all: validate test bench

validate:
	golangci-lint run

clean:
	find . -name flymake_* -delete

test: clean
	go test -v .

cover: clean
	go test -v .  -coverprofile=/tmp/coverage.out
	go tool cover -html=/tmp/coverage.out

bench: clean
	go test . -test.bench=.

prof: clean
	go test . -test.bench=-. -test.memprofile=/tmp/memprof.prof -test.cpuprofile=/tmp/cpuprof.prof

sloccount:
	 find . -name "*.go" -print0 | xargs -0 wc -l
