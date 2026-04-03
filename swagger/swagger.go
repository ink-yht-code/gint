// Copyright 2025 ink-yht-code
//
// Proprietary License
//
// IMPORTANT: This software is NOT open source.
// You may NOT use, copy, modify, merge, publish, distribute, sublicense,
// or sell copies of this file, in whole or in part, without prior written
// permission from the copyright holder.
//
// This software is provided "AS IS", without warranty of any kind.

package swagger

import (
	"encoding/json"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Config Swagger 文档配置
type Config struct {
	Title          string
	Description    string
	Version        string
	TermsOfService string
	Contact        struct {
		Name  string
		Email string
		URL   string
	}
	License struct {
		Name string
		URL  string
	}
}

// Info OpenAPI Info
type Info struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	Version        string `json:"version"`
	TermsOfService string `json:"termsOfService,omitempty"`
	Contact        struct {
		Name  string `json:"name,omitempty"`
		Email string `json:"email,omitempty"`
		URL   string `json:"url,omitempty"`
	} `json:"contact,omitempty"`
	License struct {
		Name string `json:"name,omitempty"`
		URL  string `json:"url,omitempty"`
	} `json:"license,omitempty"`
}

// Server OpenAPI Server
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem OpenAPI PathItem
type PathItem struct {
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Deprecated  bool                `json:"deprecated,omitempty"`
}

// Parameter OpenAPI Parameter
type Parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // query, header, path, cookie
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Deprecated  bool        `json:"deprecated,omitempty"`
	Schema      *Schema     `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// RequestBody OpenAPI RequestBody
type RequestBody struct {
	Description string             `json:"description,omitempty"`
	Required    bool               `json:"required"`
	Content     map[string]Content `json:"content"`
}

// Content OpenAPI Content
type Content struct {
	Schema   *Schema     `json:"schema"`
	Example  interface{} `json:"example,omitempty"`
	Examples []Example   `json:"examples,omitempty"`
}

// Example OpenAPI Example
type Example struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

// Response OpenAPI Response
type Response struct {
	Description string             `json:"description"`
	Headers     map[string]Header  `json:"headers,omitempty"`
	Content     map[string]Content `json:"content,omitempty"`
}

// Header OpenAPI Header
type Header struct {
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required"`
	Deprecated  bool    `json:"deprecated,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Schema OpenAPI Schema
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty"`
	Default              interface{}        `json:"default,omitempty"`
	Example              interface{}        `json:"example,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
}

// Document OpenAPI Document
type Document struct {
	OpenAPI    string                         `json:"openapi"`
	Info       Info                           `json:"info"`
	Servers    []Server                       `json:"servers,omitempty"`
	Paths      map[string]map[string]PathItem `json:"paths"`
	Components Components                     `json:"components,omitempty"`
	Tags       []Tag                          `json:"tags,omitempty"`
}

// Components OpenAPI Components
type Components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme OpenAPI SecurityScheme
type SecurityScheme struct {
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

// Tag OpenAPI Tag
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Builder Swagger 文档构建器
type Builder struct {
	doc      Document
	schemas  map[string]*Schema
	security []map[string][]string
}

// NewBuilder 创建 Swagger 构建器
func NewBuilder() *Builder {
	return &Builder{
		doc: Document{
			OpenAPI: "3.0.3",
			Info: Info{
				Title:   "API",
				Version: "1.0.0",
			},
			Paths:      make(map[string]map[string]PathItem),
			Components: Components{Schemas: make(map[string]*Schema)},
		},
		schemas: make(map[string]*Schema),
	}
}

// Title 设置标题
func (b *Builder) Title(title string) *Builder {
	b.doc.Info.Title = title
	return b
}

// Description 设置描述
func (b *Builder) Description(desc string) *Builder {
	b.doc.Info.Description = desc
	return b
}

// Version 设置版本
func (b *Builder) Version(version string) *Builder {
	b.doc.Info.Version = version
	return b
}

// Server 添加服务器
func (b *Builder) Server(url, description string) *Builder {
	b.doc.Servers = append(b.doc.Servers, Server{URL: url, Description: description})
	return b
}

// Tag 添加标签
func (b *Builder) Tag(name, description string) *Builder {
	b.doc.Tags = append(b.doc.Tags, Tag{Name: name, Description: description})
	return b
}

// BasicAuth 添加 Basic 认证
func (b *Builder) BasicAuth(name string) *Builder {
	if b.doc.Components.SecuritySchemes == nil {
		b.doc.Components.SecuritySchemes = make(map[string]SecurityScheme)
	}
	b.doc.Components.SecuritySchemes[name] = SecurityScheme{
		Type:   "http",
		Scheme: "basic",
	}
	return b
}

// BearerAuth 添加 Bearer 认证
func (b *Builder) BearerAuth(name string) *Builder {
	if b.doc.Components.SecuritySchemes == nil {
		b.doc.Components.SecuritySchemes = make(map[string]SecurityScheme)
	}
	b.doc.Components.SecuritySchemes[name] = SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}
	return b
}

// ApiKeyAuth 添加 API Key 认证
func (b *Builder) ApiKeyAuth(name, in, keyName string) *Builder {
	if b.doc.Components.SecuritySchemes == nil {
		b.doc.Components.SecuritySchemes = make(map[string]SecurityScheme)
	}
	b.doc.Components.SecuritySchemes[name] = SecurityScheme{
		Type: "apiKey",
		Name: keyName,
		In:   in, // header, query, cookie
	}
	return b
}

// Path 添加路径
func (b *Builder) Path(method, path string, item PathItem) *Builder {
	if b.doc.Paths[path] == nil {
		b.doc.Paths[path] = make(map[string]PathItem)
	}
	b.doc.Paths[path][strings.ToLower(method)] = item
	return b
}

// GET 添加 GET 路径
func (b *Builder) GET(path string, item PathItem) *Builder {
	return b.Path("GET", path, item)
}

// POST 添加 POST 路径
func (b *Builder) POST(path string, item PathItem) *Builder {
	return b.Path("POST", path, item)
}

// PUT 添加 PUT 路径
func (b *Builder) PUT(path string, item PathItem) *Builder {
	return b.Path("PUT", path, item)
}

// DELETE 添加 DELETE 路径
func (b *Builder) DELETE(path string, item PathItem) *Builder {
	return b.Path("DELETE", path, item)
}

// PATCH 添加 PATCH 路径
func (b *Builder) PATCH(path string, item PathItem) *Builder {
	return b.Path("PATCH", path, item)
}

// Schema 从 Go 类型生成 Schema
func (b *Builder) Schema(typ interface{}, name string) *Schema {
	return b.schemaFromType(reflect.TypeOf(typ), name)
}

// schemaFromType 从 reflect.Type 生成 Schema
func (b *Builder) schemaFromType(t reflect.Type, name string) *Schema {
	// 处理指针
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &Schema{}

	// 注册命名类型到 Components
	if name != "" && t.Name() != "" {
		if existing, ok := b.schemas[t.Name()]; ok {
			return existing
		}
		b.schemas[t.Name()] = schema
		b.doc.Components.Schemas[t.Name()] = schema
		schema.Ref = "#/components/schemas/" + t.Name()
		return schema
	}

	switch t.Kind() {
	case reflect.String:
		schema.Type = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
		schema.Minimum = ptrFloat64(0)
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		schema.Items = b.schemaFromType(t.Elem(), "")
	case reflect.Map:
		schema.Type = "object"
		schema.AdditionalProperties = b.schemaFromType(t.Elem(), "")
	case reflect.Struct:
		schema.Type = "object"
		schema.Properties = make(map[string]*Schema)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}
			fieldName := jsonTag
			if fieldName == "" {
				fieldName = field.Name
			} else {
				// 处理 json:"name,omitempty"
				fieldName = strings.Split(fieldName, ",")[0]
			}

			propSchema := b.schemaFromType(field.Type, "")

			// 从 tag 提取验证信息
			if binding := field.Tag.Get("binding"); binding != "" {
				b.applyBinding(schema, propSchema, fieldName, binding)
			}

			schema.Properties[fieldName] = propSchema
		}
	default:
		schema.Type = "object"
	}

	return schema
}

// applyBinding 应用 binding 标签到 Schema
func (b *Builder) applyBinding(parent, schema *Schema, fieldName, binding string) {
	rules := strings.Split(binding, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		switch {
		case rule == "required":
			parent.Required = append(parent.Required, fieldName)
		case strings.HasPrefix(rule, "min="):
			// handled by validator
		case strings.HasPrefix(rule, "max="):
			// handled by validator
		case rule == "email":
			schema.Format = "email"
		case rule == "url":
			schema.Format = "uri"
		}
	}
}

// Build 构建 Document
func (b *Builder) Build() Document {
	return b.doc
}

// Handler 返回 Swagger JSON 处理器
func (b *Builder) Handler() gin.HandlerFunc {
	doc := b.Build()
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, doc)
	}
}

// UI 返回 Swagger UI 处理器
func (b *Builder) UI(specURL string) gin.HandlerFunc {
	html := swaggerUIHTML(specURL)
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// --- 辅助函数 ---

func ptrInt(v int) *int             { return &v }
func ptrFloat64(v float64) *float64 { return &v }

// swaggerUIHTML 返回 Swagger UI HTML
func swaggerUIHTML(specURL string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>html, body { margin: 0; padding: 0; }</style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
    window.onload = function() {
        const ui = SwaggerUIBundle({
            url: "` + specURL + `",
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            layout: "StandaloneLayout"
        })
    }
    </script>
</body>
</html>`
}

// --- 快捷响应 ---

// OKResponse 创建 200 响应
func OKResponse(description string, schema *Schema) Response {
	return Response{
		Description: description,
		Content: map[string]Content{
			"application/json": {Schema: schema},
		},
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(code int, description string) Response {
	return Response{
		Description: description,
		Content: map[string]Content{
			"application/json": {
				Schema: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"code": {Type: "integer"},
						"msg":  {Type: "string"},
					},
				},
			},
		},
	}
}

// JSONBody 创建 JSON 请求体
func JSONBody(schema *Schema, required bool) *RequestBody {
	return &RequestBody{
		Required: required,
		Content: map[string]Content{
			"application/json": {Schema: schema},
		},
	}
}

// QueryParam 创建查询参数
func QueryParam(name, description string, required bool, schema *Schema) Parameter {
	if schema == nil {
		schema = &Schema{Type: "string"}
	}
	return Parameter{
		Name:        name,
		In:          "query",
		Description: description,
		Required:    required,
		Schema:      schema,
	}
}

// PathParam 创建路径参数
func PathParam(name, description string, schema *Schema) Parameter {
	if schema == nil {
		schema = &Schema{Type: "string"}
	}
	return Parameter{
		Name:        name,
		In:          "path",
		Description: description,
		Required:    true,
		Schema:      schema,
	}
}

// HeaderParam 创建请求头参数
func HeaderParam(name, description string, required bool) Parameter {
	return Parameter{
		Name:        name,
		In:          "header",
		Description: description,
		Required:    required,
		Schema:      &Schema{Type: "string"},
	}
}

// --- 自动注册 ---

// RouteInfo 路由信息
type RouteInfo struct {
	Method      string
	Path        string
	Handler     string
	Description string
	Tags        []string
}

// AutoRegister 自动注册 Gin 路由
func (b *Builder) AutoRegister(r *gin.Engine) {
	routes := r.Routes()
	for _, route := range routes {
		// 跳过已注册的 swagger 路由
		if strings.HasPrefix(route.Path, "/swagger") {
			continue
		}

		method := strings.ToLower(route.Method)
		path := route.Path

		// 提取路径参数
		params := extractPathParams(path)

		item := PathItem{
			Summary: route.Handler,
			Tags:    []string{"default"},
			Responses: map[string]Response{
				"200": OKResponse("成功", nil),
			},
		}

		// 添加路径参数
		for _, p := range params {
			item.Parameters = append(item.Parameters, PathParam(p, p+" 参数", nil))
		}

		if b.doc.Paths[path] == nil {
			b.doc.Paths[path] = make(map[string]PathItem)
		}
		b.doc.Paths[path][method] = item
	}
}

// extractPathParams 提取路径参数
func extractPathParams(path string) []string {
	re := regexp.MustCompile(`:([^/]+)`)
	matches := re.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, m := range matches {
		params = append(params, m[1])
	}
	return params
}

// --- 全局 Swagger ---

var defaultBuilder *Builder

// Init 初始化默认 Swagger
func Init(title, version string) *Builder {
	defaultBuilder = NewBuilder().Title(title).Version(version)
	return defaultBuilder
}

// InitWithConfig 使用 Config 初始化默认 Swagger
func InitWithConfig(cfg Config) *Builder {
	b := NewBuilder().
		Title(cfg.Title).
		Description(cfg.Description).
		Version(cfg.Version)

	if cfg.TermsOfService != "" {
		b.doc.Info.TermsOfService = cfg.TermsOfService
	}
	if cfg.Contact.Name != "" || cfg.Contact.Email != "" || cfg.Contact.URL != "" {
		b.doc.Info.Contact.Name = cfg.Contact.Name
		b.doc.Info.Contact.Email = cfg.Contact.Email
		b.doc.Info.Contact.URL = cfg.Contact.URL
	}
	if cfg.License.Name != "" || cfg.License.URL != "" {
		b.doc.Info.License.Name = cfg.License.Name
		b.doc.Info.License.URL = cfg.License.URL
	}

	defaultBuilder = b
	return defaultBuilder
}

// Default 获取默认构建器
func Default() *Builder {
	if defaultBuilder == nil {
		defaultBuilder = NewBuilder()
	}
	return defaultBuilder
}

// Register 注册 Swagger 路由。
//
// 默认注册：
// - GET /swagger.json
// - GET /swagger/ui
// - GET /swagger/index.html（兼容旧文档/习惯用法，等价于 /swagger/ui）
//
// 注意：Register 不会自动 AutoRegister 路由；如果需要自动收集路由，请使用 Setup。
func Register(r *gin.Engine) *Builder {
	b := Default()

	r.GET("/swagger.json", b.Handler())
	r.GET("/swagger/ui", b.UI("/swagger.json"))
	r.GET("/swagger/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/ui")
	})

	return b
}

// Setup 快速设置 Swagger
// 自动注册 /swagger.json 和 /swagger/ui
func Setup(r *gin.Engine, title, version string) *Builder {
	b := Init(title, version)
	b.AutoRegister(r)

	r.GET("/swagger.json", b.Handler())
	r.GET("/swagger/ui", b.UI("/swagger.json"))
	r.GET("/swagger/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/ui")
	})

	return b
}

// Marshal 将 Document 转换为 JSON
func Marshal(doc Document) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}
