GRPCF_PLUGIN=protoc-gen-grpc-funcs
EXAMPLES_GO=examples/bookstore.pb.go
EXAMPLES_PROTO=examples/bookstore.proto
EXAMPLES_FUNCS_GO=examples/bookstore.grpcf.go

$(EXAMPLES_GRPCF_GO): $(EXAMPLES_GO)
	protoc --grpc-funcs_out=. $(EXAMPLES_PROTO)
	goimports -w $(EXAMPLES_FUNCS_GO)

$(EXAMPLES_GO): $(GRPCF_PLUGIN) install
	protoc --go_out=plugins=grpc:. $(EXAMPLES_PROTO)

$(GRPCF_PLUGIN): main.go
	go build -o $@

clean:
	go clean
	rm -rf $(GRPC_PLUGIN) $(EXAMPLES_GO)

install: $(GRPC_PLUGIN)
	go install

.PHONY: install
