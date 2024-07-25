
.PHONY: cleanup-builds
cleanup-builds:
	rm -rf build
	mkdir build

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -o build/github-skyline-linux .

.PHONY: build-linux-arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags "-w -s" -o build/github-skyline-linux-arm64 .

.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "-w -s" -o build/github-skyline-macos .

.PHONY: build-darwin-arm64
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "-w -s" -o build/github-skyline-macos-arm64 .

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "-w -s" -o build/github-skyline.exe .

.PHONY: build-windows-arm64
build-windows-arm64:
	GOOS=windows GOARCH=arm64 go build -ldflags "-w -s" -o build/github-skyline-arm64.exe .

.PHONY: build
build: cleanup-builds build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows build-windows-arm64

github-release: build
	gh release create --generate-notes v$(shell ./build/github-skyline-linux --version-raw) ./build/*
