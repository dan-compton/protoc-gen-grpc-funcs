# protoc-gen-grpc-funcs

This plugin, adapted from [protoc-gen-grpc-go-service](https://github.com/nstogner/protoc-gen-grpc-go-service), generates a default implementation of grpc-defined services. This generated implementation consists of a struct whose members are functions which implement the defined rpc endpoints.  This allows the developer to implement the grpc spec one endpoint at a time, and makes the addition of mock endpoints as simple as defining a function, and pointing the struct member to that function definition.

NOTE:
* The generated implementation is meant to live in the same package as the generated grpc service interface.
* Ensure that you run `gofmt -w` on your generated code following plugin execution


# Example

A generated service implementation may look like the following. Note that a constructor function is created, as is a method for registering the implementation with a grpc server.  See the [examples](examples/) for more an more details.

```
// ExampleServiceFuncs is an implementation of the grpc-defined interface, ExampleService.
// Its members are functions which implement the defined rpc endpoints.
type ExampleServiceFuncs struct {
	EchoMethod func(ctx context.Context, input *InputMessage) (*OutputMessage, error)
}

// Echo calls the provided implementation, EchoMethod.
func (t *ExampleServiceFuncs) Echo(ctx context.Context, input *InputMessage) (*OutputMessage, error) {
	return t.EchoMethod(ctx, input)
}

// Register associates the implementation with a grpc server.
func (t *ExampleServiceFuncs) Register(srv *grpc.Server) {
	RegisterExampleServiceServer(srv, t)
}


// NewExampleServiceFuncs creates an instance of ExampleService with unimplemented method stubs.
// NOTE: you should provide your own functions which implement the underlying methods.
func NewExampleServiceFuncs() *ExampleServiceFuncs {
	var t = new(ExampleServiceFuncs)

	t.EchoMethod = func(ctx context.Context, input *InputMessage) (*OutputMessage, error) {
		return nil, status.Errorf(codes.Unimplemented, "Echo has not been implemented")
	}
    return t
}
```


# Building

Ensure that grpc/protobuf dependencies are installed. Then run `make` to build the example.
