// Code generated by protoc-gen-go-http. DO NOT EDIT.
// versions:
// - protoc-gen-go-http v2.8.2
// - protoc             (unknown)
// source: kessel/relations/v1/health.proto

package v1

import (
	context "context"
	http "github.com/go-kratos/kratos/v2/transport/http"
	binding "github.com/go-kratos/kratos/v2/transport/http/binding"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
var _ = new(context.Context)
var _ = binding.EncodeURL

const _ = http.SupportPackageIsVersion1

const OperationKesselRelationsHealthServiceGetLivez = "/kessel.relations.v1.KesselRelationsHealthService/GetLivez"
const OperationKesselRelationsHealthServiceGetReadyz = "/kessel.relations.v1.KesselRelationsHealthService/GetReadyz"

type KesselRelationsHealthServiceHTTPServer interface {
	GetLivez(context.Context, *GetLivezRequest) (*GetLivezResponse, error)
	GetReadyz(context.Context, *GetReadyzRequest) (*GetReadyzResponse, error)
}

func RegisterKesselRelationsHealthServiceHTTPServer(s *http.Server, srv KesselRelationsHealthServiceHTTPServer) {
	r := s.Route("/")
	r.GET("/livez", _KesselRelationsHealthService_GetLivez0_HTTP_Handler(srv))
	r.GET("/readyz", _KesselRelationsHealthService_GetReadyz0_HTTP_Handler(srv))
}

func _KesselRelationsHealthService_GetLivez0_HTTP_Handler(srv KesselRelationsHealthServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in GetLivezRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationKesselRelationsHealthServiceGetLivez)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.GetLivez(ctx, req.(*GetLivezRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*GetLivezResponse)
		return ctx.Result(200, reply)
	}
}

func _KesselRelationsHealthService_GetReadyz0_HTTP_Handler(srv KesselRelationsHealthServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in GetReadyzRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationKesselRelationsHealthServiceGetReadyz)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.GetReadyz(ctx, req.(*GetReadyzRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*GetReadyzResponse)
		return ctx.Result(200, reply)
	}
}

type KesselRelationsHealthServiceHTTPClient interface {
	GetLivez(ctx context.Context, req *GetLivezRequest, opts ...http.CallOption) (rsp *GetLivezResponse, err error)
	GetReadyz(ctx context.Context, req *GetReadyzRequest, opts ...http.CallOption) (rsp *GetReadyzResponse, err error)
}

type KesselRelationsHealthServiceHTTPClientImpl struct {
	cc *http.Client
}

func NewKesselRelationsHealthServiceHTTPClient(client *http.Client) KesselRelationsHealthServiceHTTPClient {
	return &KesselRelationsHealthServiceHTTPClientImpl{client}
}

func (c *KesselRelationsHealthServiceHTTPClientImpl) GetLivez(ctx context.Context, in *GetLivezRequest, opts ...http.CallOption) (*GetLivezResponse, error) {
	var out GetLivezResponse
	pattern := "/livez"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationKesselRelationsHealthServiceGetLivez))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "GET", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *KesselRelationsHealthServiceHTTPClientImpl) GetReadyz(ctx context.Context, in *GetReadyzRequest, opts ...http.CallOption) (*GetReadyzResponse, error) {
	var out GetReadyzResponse
	pattern := "/readyz"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationKesselRelationsHealthServiceGetReadyz))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "GET", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
