// Code generated by protoc-gen-go-http. DO NOT EDIT.
// versions:
// - protoc-gen-go-http v2.8.0
// - protoc             (unknown)
// source: kessel/relations/v1beta1/relation_tuples.proto

package v1beta1

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

const OperationKesselTupleServiceCreateTuples = "/kessel.relations.v1beta1.KesselTupleService/CreateTuples"
const OperationKesselTupleServiceDeleteTuples = "/kessel.relations.v1beta1.KesselTupleService/DeleteTuples"

type KesselTupleServiceHTTPServer interface {
	CreateTuples(context.Context, *CreateTuplesRequest) (*CreateTuplesResponse, error)
	DeleteTuples(context.Context, *DeleteTuplesRequest) (*DeleteTuplesResponse, error)
}

func RegisterKesselTupleServiceHTTPServer(s *http.Server, srv KesselTupleServiceHTTPServer) {
	r := s.Route("/")
	r.POST("/v1beta1/tuples", _KesselTupleService_CreateTuples0_HTTP_Handler(srv))
	r.DELETE("/v1beta1/tuples", _KesselTupleService_DeleteTuples0_HTTP_Handler(srv))
}

func _KesselTupleService_CreateTuples0_HTTP_Handler(srv KesselTupleServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in CreateTuplesRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationKesselTupleServiceCreateTuples)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.CreateTuples(ctx, req.(*CreateTuplesRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*CreateTuplesResponse)
		return ctx.Result(200, reply)
	}
}

func _KesselTupleService_DeleteTuples0_HTTP_Handler(srv KesselTupleServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in DeleteTuplesRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationKesselTupleServiceDeleteTuples)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DeleteTuples(ctx, req.(*DeleteTuplesRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*DeleteTuplesResponse)
		return ctx.Result(200, reply)
	}
}

type KesselTupleServiceHTTPClient interface {
	CreateTuples(ctx context.Context, req *CreateTuplesRequest, opts ...http.CallOption) (rsp *CreateTuplesResponse, err error)
	DeleteTuples(ctx context.Context, req *DeleteTuplesRequest, opts ...http.CallOption) (rsp *DeleteTuplesResponse, err error)
}

type KesselTupleServiceHTTPClientImpl struct {
	cc *http.Client
}

func NewKesselTupleServiceHTTPClient(client *http.Client) KesselTupleServiceHTTPClient {
	return &KesselTupleServiceHTTPClientImpl{client}
}

func (c *KesselTupleServiceHTTPClientImpl) CreateTuples(ctx context.Context, in *CreateTuplesRequest, opts ...http.CallOption) (*CreateTuplesResponse, error) {
	var out CreateTuplesResponse
	pattern := "/v1beta1/tuples"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationKesselTupleServiceCreateTuples))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *KesselTupleServiceHTTPClientImpl) DeleteTuples(ctx context.Context, in *DeleteTuplesRequest, opts ...http.CallOption) (*DeleteTuplesResponse, error) {
	var out DeleteTuplesResponse
	pattern := "/v1beta1/tuples"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationKesselTupleServiceDeleteTuples))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "DELETE", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
