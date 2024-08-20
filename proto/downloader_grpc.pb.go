// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: downloader.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	DownloadService_Init_FullMethodName          = "/DownloadService/Init"
	DownloadService_Shutdown_FullMethodName      = "/DownloadService/Shutdown"
	DownloadService_GetVideoInfo_FullMethodName  = "/DownloadService/GetVideoInfo"
	DownloadService_ParseEpisodes_FullMethodName = "/DownloadService/ParseEpisodes"
	DownloadService_Download_FullMethodName      = "/DownloadService/Download"
	DownloadService_StopDownload_FullMethodName  = "/DownloadService/StopDownload"
)

// DownloadServiceClient is the client API for DownloadService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DownloadServiceClient interface {
	Init(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error)
	Shutdown(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// ShowInfo sends metadata about the downloadable content.
	GetVideoInfo(ctx context.Context, in *VideoInfoRequest, opts ...grpc.CallOption) (*VideoInfoResponse, error)
	ParseEpisodes(ctx context.Context, in *ParseRequest, opts ...grpc.CallOption) (*ParseResponse, error)
	// Download starts the download process and streams download progress.
	Download(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[DownloadProgress], error)
	// StopDownload stops an ongoing download.
	StopDownload(ctx context.Context, in *StopDownloadRequest, opts ...grpc.CallOption) (*StopDownloadResponse, error)
}

type downloadServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDownloadServiceClient(cc grpc.ClientConnInterface) DownloadServiceClient {
	return &downloadServiceClient{cc}
}

func (c *downloadServiceClient) Init(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, DownloadService_Init_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *downloadServiceClient) Shutdown(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, DownloadService_Shutdown_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *downloadServiceClient) GetVideoInfo(ctx context.Context, in *VideoInfoRequest, opts ...grpc.CallOption) (*VideoInfoResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VideoInfoResponse)
	err := c.cc.Invoke(ctx, DownloadService_GetVideoInfo_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *downloadServiceClient) ParseEpisodes(ctx context.Context, in *ParseRequest, opts ...grpc.CallOption) (*ParseResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ParseResponse)
	err := c.cc.Invoke(ctx, DownloadService_ParseEpisodes_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *downloadServiceClient) Download(ctx context.Context, in *DownloadRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[DownloadProgress], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &DownloadService_ServiceDesc.Streams[0], DownloadService_Download_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[DownloadRequest, DownloadProgress]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type DownloadService_DownloadClient = grpc.ServerStreamingClient[DownloadProgress]

func (c *downloadServiceClient) StopDownload(ctx context.Context, in *StopDownloadRequest, opts ...grpc.CallOption) (*StopDownloadResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StopDownloadResponse)
	err := c.cc.Invoke(ctx, DownloadService_StopDownload_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DownloadServiceServer is the server API for DownloadService service.
// All implementations must embed UnimplementedDownloadServiceServer
// for forward compatibility.
type DownloadServiceServer interface {
	Init(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	Shutdown(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	// ShowInfo sends metadata about the downloadable content.
	GetVideoInfo(context.Context, *VideoInfoRequest) (*VideoInfoResponse, error)
	ParseEpisodes(context.Context, *ParseRequest) (*ParseResponse, error)
	// Download starts the download process and streams download progress.
	Download(*DownloadRequest, grpc.ServerStreamingServer[DownloadProgress]) error
	// StopDownload stops an ongoing download.
	StopDownload(context.Context, *StopDownloadRequest) (*StopDownloadResponse, error)
	mustEmbedUnimplementedDownloadServiceServer()
}

// UnimplementedDownloadServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedDownloadServiceServer struct{}

func (UnimplementedDownloadServiceServer) Init(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Init not implemented")
}
func (UnimplementedDownloadServiceServer) Shutdown(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}
func (UnimplementedDownloadServiceServer) GetVideoInfo(context.Context, *VideoInfoRequest) (*VideoInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVideoInfo not implemented")
}
func (UnimplementedDownloadServiceServer) ParseEpisodes(context.Context, *ParseRequest) (*ParseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ParseEpisodes not implemented")
}
func (UnimplementedDownloadServiceServer) Download(*DownloadRequest, grpc.ServerStreamingServer[DownloadProgress]) error {
	return status.Errorf(codes.Unimplemented, "method Download not implemented")
}
func (UnimplementedDownloadServiceServer) StopDownload(context.Context, *StopDownloadRequest) (*StopDownloadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StopDownload not implemented")
}
func (UnimplementedDownloadServiceServer) mustEmbedUnimplementedDownloadServiceServer() {}
func (UnimplementedDownloadServiceServer) testEmbeddedByValue()                         {}

// UnsafeDownloadServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DownloadServiceServer will
// result in compilation errors.
type UnsafeDownloadServiceServer interface {
	mustEmbedUnimplementedDownloadServiceServer()
}

func RegisterDownloadServiceServer(s grpc.ServiceRegistrar, srv DownloadServiceServer) {
	// If the following call pancis, it indicates UnimplementedDownloadServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&DownloadService_ServiceDesc, srv)
}

func _DownloadService_Init_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DownloadServiceServer).Init(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DownloadService_Init_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DownloadServiceServer).Init(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _DownloadService_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DownloadServiceServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DownloadService_Shutdown_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DownloadServiceServer).Shutdown(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _DownloadService_GetVideoInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VideoInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DownloadServiceServer).GetVideoInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DownloadService_GetVideoInfo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DownloadServiceServer).GetVideoInfo(ctx, req.(*VideoInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DownloadService_ParseEpisodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ParseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DownloadServiceServer).ParseEpisodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DownloadService_ParseEpisodes_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DownloadServiceServer).ParseEpisodes(ctx, req.(*ParseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DownloadService_Download_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(DownloadRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DownloadServiceServer).Download(m, &grpc.GenericServerStream[DownloadRequest, DownloadProgress]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type DownloadService_DownloadServer = grpc.ServerStreamingServer[DownloadProgress]

func _DownloadService_StopDownload_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopDownloadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DownloadServiceServer).StopDownload(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DownloadService_StopDownload_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DownloadServiceServer).StopDownload(ctx, req.(*StopDownloadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DownloadService_ServiceDesc is the grpc.ServiceDesc for DownloadService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DownloadService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "DownloadService",
	HandlerType: (*DownloadServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Init",
			Handler:    _DownloadService_Init_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _DownloadService_Shutdown_Handler,
		},
		{
			MethodName: "GetVideoInfo",
			Handler:    _DownloadService_GetVideoInfo_Handler,
		},
		{
			MethodName: "ParseEpisodes",
			Handler:    _DownloadService_ParseEpisodes_Handler,
		},
		{
			MethodName: "StopDownload",
			Handler:    _DownloadService_StopDownload_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Download",
			Handler:       _DownloadService_Download_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "downloader.proto",
}
