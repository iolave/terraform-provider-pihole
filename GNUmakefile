default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run ./...

generate:
	#cd tools; go generate ./...
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.21.0 && \
		cat CHANGELOG.md >> ./docs/index.md
	

fmt:
	gofmt -s -w -e .

test:
	go test ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

docker-run:
	docker compose up -d --build

.PHONY: fmt lint test testacc build install generate
