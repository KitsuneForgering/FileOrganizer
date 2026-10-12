.PHONY: build test run clean install

build:
	mkdir -p build/bin
	go build -o build/bin/fileorganizer-core ./cmd/fileorganizer-core
	cmake -S . -B build/gui -G Ninja
	cmake --build build/gui

test: build
	go test ./...
	ctest --test-dir build/gui --output-on-failure

run: build
	./build/bin/fileorganizer

install: build
	install -Dm755 build/bin/fileorganizer $(HOME)/.local/bin/fileorganizer
	install -Dm755 build/bin/fileorganizer-core $(HOME)/.local/bin/fileorganizer-core

clean:
	rm -rf build
