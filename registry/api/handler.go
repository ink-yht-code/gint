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

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/registry/store"
)

// Handler API 处理器
type Handler struct {
	store store.Store
}

// NewHandler 创建 Handler
func NewHandler(store store.Store) *Handler {
	return &Handler{store: store}
}

// AllocateRequest 分配请求
type AllocateRequest struct {
	Name string `json:"name" binding:"required"`
}

// AllocateResponse 分配响应
type AllocateResponse struct {
	ServiceID int    `json:"service_id"`
	Name      string `json:"name"`
}

// Allocate 分配 ServiceID
// POST /v1/services:allocate
func (h *Handler) Allocate(c *gin.Context) {
	var req AllocateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	svc, err := h.store.Allocate(req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, AllocateResponse{
		ServiceID: svc.ServiceID,
		Name:      svc.Name,
	})
}

// Get 获取服务
// GET /v1/services/:name
func (h *Handler) Get(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	svc, err := h.store.Get(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}

	c.JSON(http.StatusOK, AllocateResponse{
		ServiceID: svc.ServiceID,
		Name:      svc.Name,
	})
}

// List 列出所有服务
// GET /v1/services
func (h *Handler) List(c *gin.Context) {
	services, err := h.store.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]AllocateResponse, len(services))
	for i, svc := range services {
		result[i] = AllocateResponse{
			ServiceID: svc.ServiceID,
			Name:      svc.Name,
		}
	}

	c.JSON(http.StatusOK, gin.H{"services": result})
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")
	{
		services := v1.Group("/services")
		{
			services.POST(":allocate", h.Allocate)
			services.GET("/:name", h.Get)
			services.GET("", h.List)
		}
	}
}
