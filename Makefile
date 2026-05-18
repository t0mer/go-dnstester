.PHONY: build build-dev build-local test lint clean package package-dev

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	BUILD_MODE=prod VERSION=$(VERSION) bash scripts/build.sh

build-dev:
	BUILD_MODE=dev VERSION=$(VERSION) bash scripts/build.sh

build-local:
	BUILD_MODE=local VERSION=$(VERSION) bash scripts/build.sh

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf dist/ web/dist/*
	echo '<!-- placeholder -->' > web/dist/index.html

package:
	BUILD_MODE=prod VERSION=$(VERSION) bash scripts/build.sh

package-dev:
	BUILD_MODE=dev VERSION=$(VERSION) bash scripts/build.sh
