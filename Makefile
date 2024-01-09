.PHONY: dev build deploy
dev:
	wrangler dev

build:
	go run github.com/syumai/workers/cmd/workers-assets-gen@v0.21.0
	tinygo build -o ./build/app.wasm -target wasm -no-debug ./...

deploy:
	wrangler deploy
