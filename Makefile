.DEFAULT_GOAL := build
.PHONY: get test cov bench deps usercorn

all: binemu

clean:
	rm -f usercorn

build: get all

# dependency targets
DEST = $(shell mkdir -p deps/build; cd deps && pwd)
FIXRPATH := touch
LIBEXT := so
OS := $(shell uname -s)
ARCH := $(shell uname -m)
# name of the environment variable
LD_ENV := LD_LIBRARY_PATH="$(LD_LIBRARY_PATH):$(DEST)/lib"

ifeq "$(OS)" "Darwin"
	LD_ENV = DYLD_LIBRARY_PATH="$(DYLD_LIBRARY_PATH):$(DEST)/lib"
	LIBEXT = dylib
# this is done to fix the path that the macOS looks for libraries
	FIXRPATH = @install_name_tool \
		-add_rpath @executable_path/lib \
		-add_rpath @executable_path/deps/lib \
		-change libunicorn.dylib @rpath/libunicorn.dylib \
		-change libunicorn.1.dylib @rpath/libunicorn.1.dylib \
		-change libunicorn.2.dylib @rpath/libunicorn.2.dylib \
		-change libcapstone.dylib @rpath/libcapstone.dylib \
		-change libcapstone.3.dylib @rpath/libcapstone.3.dylib \
		-change libcapstone.4.dylib @rpath/libcapstone.4.dylib
endif

deps/lib/libunicorn.1.$(LIBEXT):
	cd deps/build && \
	git clone https://github.com/unicorn-engine/unicorn.git && git --git-dir unicorn fetch; \
	cd unicorn && git clean -fdx && git reset --hard origin/master && \
	make && make PREFIX=$(DEST) install

deps/lib/libcapstone.3.$(LIBEXT):
	cd deps/build && \
	git clone https://github.com/aquynh/capstone.git && git --git-dir capstone pull; \
	cd capstone && git clean -fdx && git reset --hard origin/master; \
	mkdir build && cd build && cmake -DCAPSTONE_BUILD_STATIC=OFF -DCMAKE_INSTALL_PREFIX=$(DEST) -DCMAKE_BUILD_TYPE=RELEASE .. && \
	make -j2 PREFIX=$(DEST) install

deps: deps/lib/libunicorn.1.$(LIBEXT) deps/lib/libcapstone.3.$(LIBEXT)

export CGO_CFLAGS = -I$(DEST)/include
export CGO_LDFLAGS = -L$(DEST)/lib

PKGS=$(shell go list .//... | sort -u | rev | sed -e 's,og/.*$$,,' | rev | sed -e 's,^,github.com/felberj/binemu/go,')

vendor: Gopkg.toml
	dep ensure
	touch vendor

.PHONY: binemu
binemu: vendor
	$(LD_ENV) go build -o binemu ./cmd/binemu
	$(FIXRPATH) binemu


usercorn: vendor
	$(LD_ENV) go build -o usercorn ./cmd/main
	$(FIXRPATH) usercorn

test: vendor
	go test -v ./...

cov: vendor
	go get -u github.com/haya14busa/goverage
	goverage -v -coverprofile=coverage.out ${PKGS}
	go tool cover -html=coverage.out

bench: vendor
	go test -v -benchmem -bench=. ./...

.PHONY: protos
protos:
	protoc -I protos/ protos/binemu.proto --go_out=proto_gen/
