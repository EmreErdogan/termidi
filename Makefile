VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o termidi .

test:
	go test ./...

# make release V=0.2.0 tags and pushes; CI builds and publishes binaries.
release:
	@test -n "$(V)" || (echo "usage: make release V=x.y.z" && exit 1)
	@grep -q "## \[$(V)\]" CHANGELOG.md || (echo "CHANGELOG.md has no [$(V)] section" && exit 1)
	git tag v$(V) && git push origin v$(V)

.PHONY: build test release
