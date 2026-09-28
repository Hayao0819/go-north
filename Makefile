GO ?= go

.PHONY: test race vet

test:
	$(GO) test ./...

race:
	CGO_ENABLED=1 $(GO) test -race ./...

vet:
	$(GO) vet ./...
