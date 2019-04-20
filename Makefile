.PHONY: update-test-data
update-test-data:
	go test -update-test-data

.PHONY: build
build:
	go get
	go build

.PHONY: test
test: build
	go test -v
