image:
	docker build -t cirocosta/l7 .

install:
	go install -v

test:
	cd ./lib && go test -v

fmt:
	go fmt
	cd ./lib && go fmt


.PHONY: install test image fmt
