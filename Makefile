.PHONY: deps css templ test test-integration run-vulnerable run-secure build clean

deps:
	go mod tidy
	npm install

css:
	npm run build:css

templ:
	go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate ./internal/views

test:
	go test ./...

test-integration:
	go test -tags=integration -count=1 -v .

run-vulnerable: css templ
	SECURITY_ENABLED=false go run .

run-secure: css templ
	SECURITY_ENABLED=true go run .

build: css templ
	go build -o app .

clean:
	rm -f app app.db coverage.out coverage.html
