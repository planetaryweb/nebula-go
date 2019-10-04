//
// plugin.go
// Copyright (C) 2019 shadow53 <shadow53@shadow53.com>
//
// Distributed under terms of the MIT license.
//

package handler

import (
	"fmt"
	"github.com/hashicorp/go-plugin"
	"gitlab.com/BluestNight/nebula-forms/errors"
	"golang.org/x/net/context"
	pb "gitlab.com/BluestNight/nebula-forms/proto"
	"google.golang.org/grpc"
	"io/ioutil"
	"net/http"
)

type Client struct {
	client pb.HandlerClient
}

func RequestToProtoRequest(r *http.Request) (*pb.HTTPRequest, error) {
	req := &pb.HTTPRequest{}
	req.Method = r.Method
	req.Url = r.URL.String()
	req.Form = make(map[string]*pb.FormValues)

	err := r.ParseMultipartForm(1024)
	if err != nil {
		return nil, err
	}

	for key, val := range r.MultipartForm.Value {
		mapVal := &pb.FormValues{}
		for _, str := range val {
			mapVal.Values = append(mapVal.Values, &pb.FormValue{
				Value: &pb.FormValue_Str{Str: str},
			})
		}
		req.Form[key] = mapVal
	}

	for key, val := range r.MultipartForm.File {
		mapVal := &pb.FormValues{}
		for _, file := range val {
			f, err := file.Open()
			if err != nil {
				return nil, err
			}
			contents, err := ioutil.ReadAll(f)
			if err != nil {
				return nil, err
			}
			mapVal.Values = append(mapVal.Values, &pb.FormValue{
				Value: &pb.FormValue_File{File: &pb.FileContents{
					FileName: file.Filename,
					Size: file.Size,
					Contents: contents,
				}},
			})
		}
		req.Form[key] = mapVal
	}

	req.Headers = make(map[string]*pb.Header)
	for key, val := range r.Header {
		head := &pb.Header{}
		for _, v :=  range val {
			head.All = append(head.All, v)
		}
		fmt.Printf("Header: %s: %#v\n", key, head.All)
		req.Headers[key] = head
	}

	return req, nil
}

func httpErrorToStatus(err *errors.HTTPError) *pb.HTTPStatus {
	if err != nil {
		return &pb.HTTPStatus{ Msg: err.Error(), Status: uint32(err.Status()) }
	}

	return &pb.HTTPStatus{ Msg: "", Status: http.StatusOK }
}

func (h *Client) Handle(req *pb.HTTPRequest) *errors.HTTPError {
	res, err := h.client.Handle(context.Background(), req)
	if err != nil {
		return errors.NewHTTPError(err.Error(), http.StatusInternalServerError)
	}

	if res.Status >= 400 {
		return errors.NewHTTPError(res.Msg, int(res.Status))
	}

	return nil
}

func (h *Client) ShouldHandle(req *pb.HTTPRequest) (bool, *errors.HTTPError) {
	res, err := h.client.ShouldHandle(context.Background(), req)
	if err != nil {
		return false, errors.NewHTTPError(err.Error(), http.StatusInternalServerError)
	}

	if res.Status.Status >= 400 {
		return false, errors.NewHTTPError(res.Status.Msg, int(res.Status.Status))
	}

	return res.Handle, nil
}

type Server struct {
	Impl Handler
}

func (s *Server) Handle(ctx context.Context, req *pb.HTTPRequest) (*pb.HTTPStatus, error) {
	if s == nil || s.Impl == nil {
		return nil, nil
	}
	return httpErrorToStatus(s.Impl.Handle(req)), nil
}

func (s *Server) ShouldHandle(ctx context.Context, req *pb.HTTPRequest) (*pb.WillHandle, error) {
	if s == nil || s.Impl == nil {
		return nil, nil
	}
	will, err := s.Impl.ShouldHandle(req)
	return &pb.WillHandle{ Handle: will, Status: httpErrorToStatus(err) }, nil
}

type Plugin struct {
	plugin.Plugin
	Impl Handler
}

func (p *Plugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterHandlerServer(s, &Server{ Impl: p.Impl })
	return nil
}

func (p *Plugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &Client{ client: pb.NewHandlerClient(c) }, nil
}