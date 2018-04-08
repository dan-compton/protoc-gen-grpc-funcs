GRPCF_PLUGIN=protoc-gen-grpc-assembly
EXAMPLES_GO=examples/examples.pb.go
EXAMPLES_PROTO=examples/examples.proto
EXAMPLES_GRPCF_GO=examples/examples.grpcf.go

$(EXAMPLES_GRPCF_GO): $(EXAMPLES_GO)
	protoc --grpc-assembly_out=. $(EXAMPLES_PROTO)


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
