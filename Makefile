PROTO_DIR := api/grpc/proto
PB_DIR := api/grpc/pb
PROTO_FILES := $(shell cd $(PROTO_DIR) 2>/dev/null && find . -type f -name '*.proto' -print)

.PHONY: proto-generate proto-check

proto-generate:
	@if [ -z "$(PROTO_FILES)" ]; then \
		echo "No protobuf contracts found in $(PROTO_DIR)"; \
	else \
		export PATH="$$(go env GOPATH)/bin:$$PATH"; \
		command -v protoc >/dev/null || { echo "protoc is required"; exit 1; }; \
		command -v protoc-gen-go >/dev/null || { echo "protoc-gen-go is required"; exit 1; }; \
		command -v protoc-gen-go-grpc >/dev/null || { echo "protoc-gen-go-grpc is required"; exit 1; }; \
		cd $(PROTO_DIR) && protoc -I=. \
			--go_out=paths=source_relative:../pb \
			--go-grpc_out=paths=source_relative:../pb \
			$(PROTO_FILES); \
	fi

proto-check: proto-generate
	@git diff --exit-code -- $(PB_DIR)
	@untracked_files="$$(git ls-files --others --exclude-standard -- $(PB_DIR) | grep -E '\.pb\.go$$' || true)"; \
	if [ -n "$$untracked_files" ]; then \
		echo "Generated protobuf files are not tracked:"; \
		echo "$$untracked_files"; \
		exit 1; \
	fi
