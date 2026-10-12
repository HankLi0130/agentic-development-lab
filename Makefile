.PHONY:verify
verify: fmt-check vet test

.PHONY:fmt
fmt:
	gofmt -w .

.PHONY:fmt-check
fmt-check:
	@out=$$(gofmt -d .); \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		exit 1; \
	fi

.PHONY:vet
vet:
	go vet ./...

.PHONY:test
test:
	go test ./...

.PHONY:run
run:
	go run .
