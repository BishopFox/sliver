Protobuf
=========

This directory contains the protobuf message definitions and is organized in the following packages:

 * `commonpb` - Common generic messages shared between `clientpb` and `sliverpb`. Notably the generic `Request` and `Response` types, which are used as headers in gRPC request/responses.
 * `clientpb` - These messages should _only_ be sent from the client to server or vice versa.
 * `sliverpb` - These message may be sent from the client to the server or from the server to the implant and vice versa. Not all messages defined in this file will appear in client<->server communication, some are specific to implant<->server.
 * `rpcpb` - gRPC service definitions
 
## Naming

The protobuf messages and service definitions follow a naming schemes.

### Unary Operations

Unary requests are named with method subject and the suffix `Req` and corresponding responses simple the method subject. For example, with `Foo` the following scheme would be used, assuming `Foo` messages are only sent between a client and the server:

```protobuf
rpc Foo(clientpb.FooReq) returns (clientpb.Foo);
```

In cases where the `Foo` message just needs to be forwarded to the implant the definition would simply put in `sliverpb` package:

```protobuf
rpc Foo(sliverpb.FooReq) returns (sliverpb.Foo);
```

In other cases the server may need to perform certain operations based on the gRPC request and send subsequent messages to the implant. For example, in the "GetSystem" workflow the server will compile a new implant based on the request's (`clientpb.GetSystemReq`) `clientpb.ImplantConfig`, which we do not want to forward to the implant. In this `sliverpb` will define a corresponding message with the prefix `Invoke` for in this case it would be `sliverpb.InvokeGetSystemReq`. `Invoke`-style messages are only sent to the implant and should not appear in gRPC definitions.
