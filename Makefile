.PHONY: build build-web dev run clean test test-all

build-web:
	cd web && pnpm install && pnpm build

build: build-web
	go build -o watch-together${EXT} .

dev:
	go run -tags dev .

run:
	./watch-together${EXT}

test:
	go test ./internal/... -v
	cd web && pnpm test

test-go:
	go test ./internal/room/... -v

test-web:
	cd web && pnpm test

clean:
	rm -f watch-together
	rm -rf web/dist

# Cross compilation
build-linux:
	$(MAKE) build EXT=
	GOOS=linux GOARCH=amd64 go build -o watch-together-linux .

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o watch-together-mac .

build-win:
	$(MAKE) build EXT=.exe
	GOOS=windows GOARCH=amd64 go build -o watch-together.exe .
