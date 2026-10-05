.PHONY: dev build build-windows test clean

dev:
	@trap 'kill 0' INT TERM; \
	(cd backend && go run ./cmd/smeditor) & \
	(cd frontend && npm run dev) & \
	wait

build:
	cd frontend && npm install && npm run build
	rm -rf backend/internal/webdist/dist
	mkdir -p backend/internal/webdist/dist
	cp -r frontend/dist/. backend/internal/webdist/dist/
	cd backend && go build -tags embed_prod -o bin/smeditor ./cmd/smeditor

# Cross-compiles smeditor.exe from Ubuntu, per docs/setup-windows.md.
# modernc.org/sqlite is pure Go (no CGO), so no mingw/cross C toolchain
# is needed to produce a Windows binary from here.
build-windows:
	cd frontend && npm install && npm run build
	rm -rf backend/internal/webdist/dist
	mkdir -p backend/internal/webdist/dist
	cp -r frontend/dist/. backend/internal/webdist/dist/
	cd backend && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags embed_prod -o bin/smeditor.exe ./cmd/smeditor

test:
	cd backend && go vet ./... && go test ./...

clean:
	rm -rf backend/bin backend/internal/webdist/dist frontend/dist
