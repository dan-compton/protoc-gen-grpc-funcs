# protoc-gen-grpc-impl(ementation)

This plugin, adapted from [protoc-gen-grpc-go-service](https://github.com/nstogner/protoc-gen-grpc-go-service), generates a default implementation of grpc-defined services. This generated implementation consists of a struct whose members are functions which implement the defined rpc endpoints.  This allows the developer to implement the grpc spec one endpoint at a time, and makes the addition of mock endpoints as simple as defining a function, and pointing the struct member to that function definition.  

See the [examples](examples/) for more details.


# Building

Ensure that grpc/protobuf dependencies are installed. Then run `make` to build the examples.
