VERSION := $(shell cat ./VERSION)

install:
	go install -v

image:
	docker build -t cirocosta/l7 .

test:
	cd ./lib && go test -v

fmt:
	go fmt
	cd ./lib && go fmt


release:
	git tag -a $(VERSION) -m "Release" || true
	git push origin $(VERSION)
	goreleaser --rm-dist

.PHONY: install test image fmt release
