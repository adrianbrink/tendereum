GOTOOLS := github.com/Masterminds/glide \
					 github.com/alecthomas/gometalinter

PACKAGES := $(shell glide novendor)

all: ensure_tools get_vendor_deps test linter install
	@echo "--> Installing tools and dependencies, running tests and linters, and installing"

install:
	@echo "--> Running go install"
	go install --ldflags '-extldflags "-static"' \
		--ldflags "-X github.com/tendermint/tendermint/version.GitCommit=`git rev-parse HEAD`" \
		./cmd/tendereum

build:
	@echo "--> Running go build --race"
	go build --ldflags '-extldflags "-static"' \
		--ldflags "-X github.com/tendermint/tendermint/version.GitCommit=`git rev-parse HEAD`" \
		-race -o build/tendereum ./cmd/tendereum

run: build
	@echo "--> Running Tendereum binary"
	./build/tendereum

test:
	@echo "--> Running go test --race"
	go test -v -race $(PACKAGES)

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

ensure_tools:
	go get $(GOTOOLS)
	gometalinter --install
