GOTOOLS := github.com/Masterminds/glide \
					 github.com/alecthomas/gometalinter \

PACKAGES := $(shell glide novendor)

all: ensure_tools get_vendor_deps test linter install_full
	@echo "--> Installing tools and dependencies, running tests and linters, and installing"

install_full:
	@echo "--> Running go install"
	go install --ldflags '-extldflags "-static"' \
		--ldflags "-X github.com/cosmos/tendereum/version.GitCommit=`git rev-parse HEAD`" \
		./cmd/full-tendereum

install_light:
	@echo "--> Running go install"
	go install --ldflags '-extldflags "-static"' \
		--ldflags "-X github.com/cosmos/tendereum/version.GitCommit=`git rev-parse HEAD`" \
		./cmd/light-tendereum


build:
	@echo "--> Running go build --race"
	rm -rf build/tendereum
	go build --ldflags '-extldflags "-static"' \
		--ldflags "-X github.com/adrianbrink/tendereum/version.GitCommit=`git rev-parse HEAD`" \
		-race -o build/tendereum ./cmd/tendereum

run: build
	@echo "--> Running Tendereum binary"
	./build/tendereum

test:
	@echo "--> Running go test -race"
	go test -v -race $(PACKAGES)

test_fast:
	@echo "--> Running go test (no race)"
	go test $(PACKAGES)

test_coverage:
	@echo "--> Running go test --race --cover"
	./devops/test_coverage.sh

test_integration:
	@echo "--> Running integration tests"
	@echo "Not yet implemented"

clean:
	@echo "--> Running clean"
	rm -rf vendor/
	rm -rf build/

linter:
	gometalinter --vendor --enable-all --tests --line-length=100 --deadline=120s ./...

get_vendor_deps:
	glide install

update_deps:
	glide up

release:
	go get github.com/goreleaser/goreleaser
	goreleaser

ensure_tools:
	go get $(GOTOOLS)
	gometalinter --install
