// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: kessel/relations/v1beta1/relation_tuples.proto

package v1beta1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	KesselTupleService_CreateTuples_FullMethodName = "/kessel.relations.v1beta1.KesselTupleService/CreateTuples"
	KesselTupleService_ReadTuples_FullMethodName   = "/kessel.relations.v1beta1.KesselTupleService/ReadTuples"
	KesselTupleService_DeleteTuples_FullMethodName = "/kessel.relations.v1beta1.KesselTupleService/DeleteTuples"
)

// KesselTupleServiceClient is the client API for KesselTupleService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// KesselTupleServices manages the persisted _Tuples_ stored in the system..
//
// A Tuple is an explicitly stated, persistent relation
// between a Resource and a Subject or Subject Set.
// It has the same _shape_ as a Relationship but is not the same thing as a Relationship.
//
// A single Tuple may result in zero-to-many Relationships.
type KesselTupleServiceClient interface {
	CreateTuples(ctx context.Context, in *CreateTuplesRequest, opts ...grpc.CallOption) (*CreateTuplesResponse, error)
	ReadTuples(ctx context.Context, in *ReadTuplesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[ReadTuplesResponse], error)
	DeleteTuples(ctx context.Context, in *DeleteTuplesRequest, opts ...grpc.CallOption) (*DeleteTuplesResponse, error)
}

type kesselTupleServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewKesselTupleServiceClient(cc grpc.ClientConnInterface) KesselTupleServiceClient {
	return &kesselTupleServiceClient{cc}
}

func (c *kesselTupleServiceClient) CreateTuples(ctx context.Context, in *CreateTuplesRequest, opts ...grpc.CallOption) (*CreateTuplesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreateTuplesResponse)
	err := c.cc.Invoke(ctx, KesselTupleService_CreateTuples_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *kesselTupleServiceClient) ReadTuples(ctx context.Context, in *ReadTuplesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[ReadTuplesResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &KesselTupleService_ServiceDesc.Streams[0], KesselTupleService_ReadTuples_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[ReadTuplesRequest, ReadTuplesResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type KesselTupleService_ReadTuplesClient = grpc.ServerStreamingClient[ReadTuplesResponse]

func (c *kesselTupleServiceClient) DeleteTuples(ctx context.Context, in *DeleteTuplesRequest, opts ...grpc.CallOption) (*DeleteTuplesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteTuplesResponse)
	err := c.cc.Invoke(ctx, KesselTupleService_DeleteTuples_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// KesselTupleServiceServer is the server API for KesselTupleService service.
// All implementations must embed UnimplementedKesselTupleServiceServer
// for forward compatibility.
//
// KesselTupleServices manages the persisted _Tuples_ stored in the system..
//
// A Tuple is an explicitly stated, persistent relation
// between a Resource and a Subject or Subject Set.
// It has the same _shape_ as a Relationship but is not the same thing as a Relationship.
//
// A single Tuple may result in zero-to-many Relationships.
type KesselTupleServiceServer interface {
	CreateTuples(context.Context, *CreateTuplesRequest) (*CreateTuplesResponse, error)
	ReadTuples(*ReadTuplesRequest, grpc.ServerStreamingServer[ReadTuplesResponse]) error
	DeleteTuples(context.Context, *DeleteTuplesRequest) (*DeleteTuplesResponse, error)
	mustEmbedUnimplementedKesselTupleServiceServer()
}

// UnimplementedKesselTupleServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedKesselTupleServiceServer struct{}

func (UnimplementedKesselTupleServiceServer) CreateTuples(context.Context, *CreateTuplesRequest) (*CreateTuplesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateTuples not implemented")
}
func (UnimplementedKesselTupleServiceServer) ReadTuples(*ReadTuplesRequest, grpc.ServerStreamingServer[ReadTuplesResponse]) error {
	return status.Errorf(codes.Unimplemented, "method ReadTuples not implemented")
}
func (UnimplementedKesselTupleServiceServer) DeleteTuples(context.Context, *DeleteTuplesRequest) (*DeleteTuplesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteTuples not implemented")
}
func (UnimplementedKesselTupleServiceServer) mustEmbedUnimplementedKesselTupleServiceServer() {}
func (UnimplementedKesselTupleServiceServer) testEmbeddedByValue()                            {}

// UnsafeKesselTupleServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to KesselTupleServiceServer will
// result in compilation errors.
type UnsafeKesselTupleServiceServer interface {
	mustEmbedUnimplementedKesselTupleServiceServer()
}

func RegisterKesselTupleServiceServer(s grpc.ServiceRegistrar, srv KesselTupleServiceServer) {
	// If the following call pancis, it indicates UnimplementedKesselTupleServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&KesselTupleService_ServiceDesc, srv)
}

func _KesselTupleService_CreateTuples_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateTuplesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KesselTupleServiceServer).CreateTuples(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: KesselTupleService_CreateTuples_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KesselTupleServiceServer).CreateTuples(ctx, req.(*CreateTuplesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _KesselTupleService_ReadTuples_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ReadTuplesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(KesselTupleServiceServer).ReadTuples(m, &grpc.GenericServerStream[ReadTuplesRequest, ReadTuplesResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type KesselTupleService_ReadTuplesServer = grpc.ServerStreamingServer[ReadTuplesResponse]

func _KesselTupleService_DeleteTuples_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteTuplesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(KesselTupleServiceServer).DeleteTuples(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: KesselTupleService_DeleteTuples_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(KesselTupleServiceServer).DeleteTuples(ctx, req.(*DeleteTuplesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// KesselTupleService_ServiceDesc is the grpc.ServiceDesc for KesselTupleService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var KesselTupleService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "kessel.relations.v1beta1.KesselTupleService",
	HandlerType: (*KesselTupleServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateTuples",
			Handler:    _KesselTupleService_CreateTuples_Handler,
		},
		{
			MethodName: "DeleteTuples",
			Handler:    _KesselTupleService_DeleteTuples_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ReadTuples",
			Handler:       _KesselTupleService_ReadTuples_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "kessel/relations/v1beta1/relation_tuples.proto",
}
