package web

import (
	"example/user/internal/server"
	"example/user/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint"
	"github.com/ink-yht-code/gint/gctx"
)

// Handler HTTP 处理器
// 实现 gint.Handler 接口
type Handler struct {
	svc *server.UserService
}

// NewHandler 创建 Handler
func NewHandler(svc *server.UserService) *Handler {
	return &Handler{svc: svc}
}

// PrivateRoutes 注册需要认证的路由
func (h *Handler) PrivateRoutes(server *gin.Engine) {
	// 需要认证的路由使用 gint.S 或 gint.BS 包装器
	// server.GET("/api/v1/profile", gint.S(h.GetProfile))
}

// PublicRoutes 注册公开的路由
func (h *Handler) PublicRoutes(server *gin.Engine) {
	// 公开路由使用 gint.W 或 gint.B 包装器
	server.POST("/api/v1/users", gint.B(h.CreateUser))
	server.GET("/api/v1/users/:id", gint.W(h.GetUser))
	server.GET("/api/v1/users", gint.B(h.ListUsers))
	server.DELETE("/api/v1/users/:id", gint.W(h.DeleteUser))
}

// CreateUser 创建用户
// 使用 gint.B 包装器自动绑定参数
func (h *Handler) CreateUser(ctx *gctx.Context, req *types.CreateUserReq) (gint.Result, error) {
	// 使用 gint.validator 进行参数校验
	vb := gint.NewValidatorBuilder()
	vb.Field("name", req.Name).AddRule(gint.Required())
	vb.Field("email", req.Email).AddRule(gint.Required()).AddRule(gint.Email())
	vb.Validate()

	if !vb.IsValid() {
		return gint.InvalidParam(vb.GetFirstError()), nil
	}

	user, err := h.svc.CreateUser(ctx.Request.Context(), req.Name, req.Email)
	if err != nil {
		return gint.InternalError(), err
	}
	return gint.Result{Code: 0, Data: &types.CreateUserResp{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
	}}, nil
}

// GetUser 获取用户
// 使用 gint.W 包装器（无参数绑定）
func (h *Handler) GetUser(ctx *gctx.Context) (gint.Result, error) {
	id := ctx.Param("id").Int64Or(0)
	if id <= 0 {
		return gint.InvalidParam("invalid id"), nil
	}
	user, err := h.svc.GetUser(ctx.Request.Context(), id)
	if err != nil {
		return gint.InternalError(), err
	}
	return gint.Result{Code: 0, Data: &types.GetUserResp{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
	}}, nil
}

// ListUsers 列出用户
// 使用 gint.B 包装器自动绑定参数
func (h *Handler) ListUsers(ctx *gctx.Context, req *types.ListUsersReq) (gint.Result, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	users, total, err := h.svc.ListUsers(ctx.Request.Context(), req.Page, req.Size)
	if err != nil {
		return gint.InternalError(), err
	}
	items := make([]types.GetUserResp, len(users))
	for i, u := range users {
		items[i] = types.GetUserResp{Id: u.Id, Name: u.Name, Email: u.Email}
	}
	return gint.Result{Code: 0, Data: &types.ListUsersResp{Total: total, Items: items}}, nil
}

// DeleteUser 删除用户
// 使用 gint.W 包装器（无参数绑定）
func (h *Handler) DeleteUser(ctx *gctx.Context) (gint.Result, error) {
	id := ctx.Param("id").Int64Or(0)
	if id <= 0 {
		return gint.InvalidParam("invalid id"), nil
	}
	if err := h.svc.DeleteUser(ctx.Request.Context(), id); err != nil {
		return gint.InternalError(), err
	}
	return gint.Result{Code: 0}, nil
}

// Health 健康检查
func (h *Handler) Health(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
