.PHONY: build test run clean install uninstall

build:
	mkdir -p build/bin
	go build -o build/bin/fileorganizer-core ./cmd/fileorganizer-core
	go build -o build/bin/fileorganizer-daemon ./cmd/fileorganizer-daemon
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
	install -Dm755 build/bin/fileorganizer-daemon $(HOME)/.local/bin/fileorganizer-daemon

uninstall:
	rm -f $(HOME)/.local/bin/fileorganizer $(HOME)/.local/bin/fileorganizer-core $(HOME)/.local/bin/fileorganizer-daemon

clean:
	rm -rf build
