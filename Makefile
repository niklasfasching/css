.PHONY: update-test-data
update-test-data:
	go test -update-test-data

.PHONY: build
build:
	go get -u ./...
	go build

.PHONY: test
test: build
	go test -v -bench=.

.PHONY: fuzz
fuzz:
	go get -u github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
	mkdir -p fuzz/
	go-fuzz-build -o=fuzz/css-fuzz.zip github.com/niklasfasching/css
	go-fuzz -bin=fuzz/css-fuzz.zip -workdir=fuzz
