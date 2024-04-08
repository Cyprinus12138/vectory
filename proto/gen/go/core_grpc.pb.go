// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.6.1
// source: proto/vectory/core.proto

package __

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// CoreClient is the client API for Core service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CoreClient interface {
	// Sends a greeting
	SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error)
	Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error)
	ListIndex(ctx context.Context, in *ListIndexRequest, opts ...grpc.CallOption) (*ListIndexResponse, error)
	CreateIndex(ctx context.Context, in *CreateIndexRequest, opts ...grpc.CallOption) (*CreateIndexResponse, error)
	GetIndexStat(ctx context.Context, in *GetIndexStatRequest, opts ...grpc.CallOption) (*GetIndexStatResponse, error)
	GetIndexMeta(ctx context.Context, in *GetIndexMetaRequest, opts ...grpc.CallOption) (*GetIndexMetaResponse, error)
}

type coreClient struct {
	cc grpc.ClientConnInterface
}

func NewCoreClient(cc grpc.ClientConnInterface) CoreClient {
	return &coreClient{cc}
}

func (c *coreClient) SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
	out := new(HelloReply)
	err := c.cc.Invoke(ctx, "/Core/SayHello", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error) {
	out := new(SearchResponse)
	err := c.cc.Invoke(ctx, "/Core/Search", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) ListIndex(ctx context.Context, in *ListIndexRequest, opts ...grpc.CallOption) (*ListIndexResponse, error) {
	out := new(ListIndexResponse)
	err := c.cc.Invoke(ctx, "/Core/ListIndex", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) CreateIndex(ctx context.Context, in *CreateIndexRequest, opts ...grpc.CallOption) (*CreateIndexResponse, error) {
	out := new(CreateIndexResponse)
	err := c.cc.Invoke(ctx, "/Core/CreateIndex", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) GetIndexStat(ctx context.Context, in *GetIndexStatRequest, opts ...grpc.CallOption) (*GetIndexStatResponse, error) {
	out := new(GetIndexStatResponse)
	err := c.cc.Invoke(ctx, "/Core/GetIndexStat", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *coreClient) GetIndexMeta(ctx context.Context, in *GetIndexMetaRequest, opts ...grpc.CallOption) (*GetIndexMetaResponse, error) {
	out := new(GetIndexMetaResponse)
	err := c.cc.Invoke(ctx, "/Core/GetIndexMeta", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CoreServer is the server API for Core service.
// All implementations must embed UnimplementedCoreServer
// for forward compatibility
type CoreServer interface {
	// Sends a greeting
	SayHello(context.Context, *HelloRequest) (*HelloReply, error)
	Search(context.Context, *SearchRequest) (*SearchResponse, error)
	ListIndex(context.Context, *ListIndexRequest) (*ListIndexResponse, error)
	CreateIndex(context.Context, *CreateIndexRequest) (*CreateIndexResponse, error)
	GetIndexStat(context.Context, *GetIndexStatRequest) (*GetIndexStatResponse, error)
	GetIndexMeta(context.Context, *GetIndexMetaRequest) (*GetIndexMetaResponse, error)
	mustEmbedUnimplementedCoreServer()
}

// UnimplementedCoreServer must be embedded to have forward compatible implementations.
type UnimplementedCoreServer struct {
}

func (UnimplementedCoreServer) SayHello(context.Context, *HelloRequest) (*HelloReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SayHello not implemented")
}
func (UnimplementedCoreServer) Search(context.Context, *SearchRequest) (*SearchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedCoreServer) ListIndex(context.Context, *ListIndexRequest) (*ListIndexResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListIndex not implemented")
}
func (UnimplementedCoreServer) CreateIndex(context.Context, *CreateIndexRequest) (*CreateIndexResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateIndex not implemented")
}
func (UnimplementedCoreServer) GetIndexStat(context.Context, *GetIndexStatRequest) (*GetIndexStatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetIndexStat not implemented")
}
func (UnimplementedCoreServer) GetIndexMeta(context.Context, *GetIndexMetaRequest) (*GetIndexMetaResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetIndexMeta not implemented")
}
func (UnimplementedCoreServer) mustEmbedUnimplementedCoreServer() {}

// UnsafeCoreServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CoreServer will
// result in compilation errors.
type UnsafeCoreServer interface {
	mustEmbedUnimplementedCoreServer()
}

func RegisterCoreServer(s grpc.ServiceRegistrar, srv CoreServer) {
	s.RegisterService(&Core_ServiceDesc, srv)
}

func _Core_SayHello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HelloRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).SayHello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Core/SayHello",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).SayHello(ctx, req.(*HelloRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Core/Search",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).Search(ctx, req.(*SearchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_ListIndex_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListIndexRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).ListIndex(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Core/ListIndex",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).ListIndex(ctx, req.(*ListIndexRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_CreateIndex_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateIndexRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).CreateIndex(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Core/CreateIndex",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).CreateIndex(ctx, req.(*CreateIndexRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_GetIndexStat_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetIndexStatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).GetIndexStat(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Core/GetIndexStat",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).GetIndexStat(ctx, req.(*GetIndexStatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Core_GetIndexMeta_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetIndexMetaRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CoreServer).GetIndexMeta(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Core/GetIndexMeta",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CoreServer).GetIndexMeta(ctx, req.(*GetIndexMetaRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Core_ServiceDesc is the grpc.ServiceDesc for Core service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Core_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Core",
	HandlerType: (*CoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _Core_SayHello_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _Core_Search_Handler,
		},
		{
			MethodName: "ListIndex",
			Handler:    _Core_ListIndex_Handler,
		},
		{
			MethodName: "CreateIndex",
			Handler:    _Core_CreateIndex_Handler,
		},
		{
			MethodName: "GetIndexStat",
			Handler:    _Core_GetIndexStat_Handler,
		},
		{
			MethodName: "GetIndexMeta",
			Handler:    _Core_GetIndexMeta_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/vectory/core.proto",
}