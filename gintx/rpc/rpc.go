// Copyright 2025 ink-yht-code
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package rpc 提供 gRPC server 初始化和拦截器
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/ink-yht-code/gint/gintx/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Config gRPC 配置
type Config struct {
	Enabled bool
	Addr    string
}

// Server gRPC 服务器
type Server struct {
	Server *grpc.Server
	addr   string
}

// NewServer 创建 gRPC 服务器
func NewServer(cfg Config) *Server {
	if !cfg.Enabled {
		return nil
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	return &Server{
		Server: s,
		addr:   cfg.Addr,
	}
}

// Run 启动服务器
func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	log.Info("gRPC server starting", zap.String("addr", s.addr))
	return s.Server.Serve(lis)
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	s.Server.GracefulStop()
	return nil
}

// unaryInterceptor 一元拦截器
func unaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 注入 request_id
	requestID := getRequestID(ctx)
	if requestID != "" {
		ctx = context.WithValue(ctx, "request_id", requestID)
	}

	// 记录请求
	log.Ctx(ctx).Info("gRPC request",
		zap.String("method", info.FullMethod),
		zap.String("peer", getPeerAddr(ctx)),
	)

	// 调用 handler
	resp, err := handler(ctx, req)

	// 错误处理
	if err != nil {
		log.Ctx(ctx).Error("gRPC error",
			zap.String("method", info.FullMethod),
			zap.Error(err),
		)
	}

	return resp, err
}

// streamInterceptor 流拦截器
func streamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()

	// 注入 request_id
	requestID := getRequestID(ctx)
	if requestID != "" {
		ctx = context.WithValue(ctx, "request_id", requestID)
	}

	// 记录请求
	log.Ctx(ctx).Info("gRPC stream",
		zap.String("method", info.FullMethod),
		zap.String("peer", getPeerAddr(ctx)),
	)

	return handler(srv, ss)
}

// getRequestID 从 metadata 获取 request_id
func getRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	ids := md.Get("x-request-id")
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// getPeerAddr 获取客户端地址
func getPeerAddr(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	return p.Addr.String()
}

// MapError 将 BizError 映射到 gRPC status
func MapError(err error) error {
	if err == nil {
		return nil
	}

	// 检查是否是 BizError
	type bizError interface {
		BizCode() int
		BizMsg() string
	}

	if biz, ok := err.(bizError); ok {
		code := mapBizCodeToGrpcCode(biz.BizCode())
		return status.Error(code, biz.BizMsg())
	}

	return status.Error(codes.Internal, err.Error())
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	BizCode   int32  `json:"biz_code"`
	BizMsg    string `json:"biz_msg"`
	RequestId string `json:"request_id"`
}

// mapBizCodeToGrpcCode 映射业务码到 gRPC 码
func mapBizCodeToGrpcCode(bizCode int) codes.Code {
	// 业务码后 4 位
	suffix := bizCode % 10000
	switch suffix {
	case 1: // InvalidParam
		return codes.InvalidArgument
	case 2: // Unauthorized
		return codes.Unauthenticated
	case 3: // Forbidden
		return codes.PermissionDenied
	case 4: // NotFound
		return codes.NotFound
	case 5: // Conflict
		return codes.AlreadyExists
	default:
		return codes.Internal
	}
}

func getRequestIDFromContext() string {
	// 从 context 获取 request_id
	return ""
}

// Client gRPC 客户端
type Client struct {
	conn *grpc.ClientConn
}

// NewClient 创建 gRPC 客户端
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// Conn 获取连接
func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.conn.Close()
}

// MarshalErrorDetail 序列化错误详情
func MarshalErrorDetail(detail *ErrorDetail) string {
	data, _ := json.Marshal(detail)
	return string(data)
}

// UnmarshalErrorDetail 反序列化错误详情
func UnmarshalErrorDetail(data string) (*ErrorDetail, error) {
	var detail ErrorDetail
	if err := json.Unmarshal([]byte(data), &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}
