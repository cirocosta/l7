install:
	go install -v

image:
	docker build -t cirocosta/l7 .

test:
	cd ./lib && go test -v

fmt:
	go fmt
	cd ./lib && go fmt


.PHONY: install test image fmt
