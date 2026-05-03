// 版权所有 2025 ink-yht-code
//
// 专有许可
//
// 重要说明：本软件并非开源软件。
// 未经版权持有人事先书面许可，
// 不得使用、复制、修改、合并、发布、分发、再许可，
// 也不得全部或部分出售本文件的副本。
//
// 本软件按“现状”提供，不附带任何形式的担保。

package swagger

import (
	"encoding/json"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Config 定义 Swagger 文档配置。
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

// Info 表示 OpenAPI 的基础信息。
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

// Server 表示 OpenAPI 服务节点。
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem 表示一个接口节点。
type PathItem struct {
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Deprecated  bool                `json:"deprecated,omitempty"`
}

// Parameter 表示接口参数。
type Parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // query、header、path、cookie
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Deprecated  bool        `json:"deprecated,omitempty"`
	Schema      *Schema     `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// RequestBody 表示请求体定义。
type RequestBody struct {
	Description string             `json:"description,omitempty"`
	Required    bool               `json:"required"`
	Content     map[string]Content `json:"content"`
}

// Content 表示内容定义。
type Content struct {
	Schema   *Schema     `json:"schema"`
	Example  interface{} `json:"example,omitempty"`
	Examples []Example   `json:"examples,omitempty"`
}

// Example 表示示例数据。
type Example struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

// Response 表示接口响应。
type Response struct {
	Description string             `json:"description"`
	Headers     map[string]Header  `json:"headers,omitempty"`
	Content     map[string]Content `json:"content,omitempty"`
}

// Header 表示响应头定义。
type Header struct {
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required"`
	Deprecated  bool    `json:"deprecated,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Schema 表示 OpenAPI 数据模型。
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

// Document 表示 OpenAPI 文档。
type Document struct {
	OpenAPI    string                         `json:"openapi"`
	Info       Info                           `json:"info"`
	Servers    []Server                       `json:"servers,omitempty"`
	Paths      map[string]map[string]PathItem `json:"paths"`
	Components Components                     `json:"components,omitempty"`
	Tags       []Tag                          `json:"tags,omitempty"`
}

// Components 表示组件定义。
type Components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme 表示安全认证方案。
type SecurityScheme struct {
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

// Tag 表示接口标签。
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Builder 用于构建 Swagger 文档。
type Builder struct {
	doc      Document
	schemas  map[string]*Schema
	security []map[string][]string
}

// NewBuilder 创建 Swagger 构建器。
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

// Title 设置文档标题。
func (b *Builder) Title(title string) *Builder {
	b.doc.Info.Title = title
	return b
}

// Description 设置文档描述。
func (b *Builder) Description(desc string) *Builder {
	b.doc.Info.Description = desc
	return b
}

// Version 设置文档版本。
func (b *Builder) Version(version string) *Builder {
	b.doc.Info.Version = version
	return b
}

// Server 添加服务地址。
func (b *Builder) Server(url, description string) *Builder {
	b.doc.Servers = append(b.doc.Servers, Server{URL: url, Description: description})
	return b
}

// Tag 添加文档标签。
func (b *Builder) Tag(name, description string) *Builder {
	b.doc.Tags = append(b.doc.Tags, Tag{Name: name, Description: description})
	return b
}

// BasicAuth 添加 Basic 认证方案。
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

// BearerAuth 添加 Bearer 认证方案。
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

// ApiKeyAuth 添加 API Key 认证方案。
func (b *Builder) ApiKeyAuth(name, in, keyName string) *Builder {
	if b.doc.Components.SecuritySchemes == nil {
		b.doc.Components.SecuritySchemes = make(map[string]SecurityScheme)
	}
	b.doc.Components.SecuritySchemes[name] = SecurityScheme{
		Type: "apiKey",
		Name: keyName,
		In:   in, // header、query、cookie
	}
	return b
}

// Path 添加一个接口路径。
func (b *Builder) Path(method, path string, item PathItem) *Builder {
	if b.doc.Paths[path] == nil {
		b.doc.Paths[path] = make(map[string]PathItem)
	}
	b.doc.Paths[path][strings.ToLower(method)] = item
	return b
}

// GET 添加 GET 接口。
func (b *Builder) GET(path string, item PathItem) *Builder {
	return b.Path("GET", path, item)
}

// POST 添加 POST 接口。
func (b *Builder) POST(path string, item PathItem) *Builder {
	return b.Path("POST", path, item)
}

// PUT 添加 PUT 接口。
func (b *Builder) PUT(path string, item PathItem) *Builder {
	return b.Path("PUT", path, item)
}

// DELETE 添加 DELETE 接口。
func (b *Builder) DELETE(path string, item PathItem) *Builder {
	return b.Path("DELETE", path, item)
}

// PATCH 添加 PATCH 接口。
func (b *Builder) PATCH(path string, item PathItem) *Builder {
	return b.Path("PATCH", path, item)
}

// Schema 根据 Go 类型生成模型定义。
func (b *Builder) Schema(typ interface{}, name string) *Schema {
	return b.schemaFromType(reflect.TypeOf(typ), name)
}

// schemaFromType 根据 reflect.Type 生成模型定义。
func (b *Builder) schemaFromType(t reflect.Type, name string) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &Schema{}

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
				fieldName = strings.Split(fieldName, ",")[0]
			}

			propSchema := b.schemaFromType(field.Type, "")

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

// applyBinding 将 binding 规则应用到模型定义。
func (b *Builder) applyBinding(parent, schema *Schema, fieldName, binding string) {
	rules := strings.Split(binding, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		switch {
		case rule == "required":
			parent.Required = append(parent.Required, fieldName)
		case strings.HasPrefix(rule, "min="):
			// 预留给验证器处理
		case strings.HasPrefix(rule, "max="):
			// 预留给验证器处理
		case rule == "email":
			schema.Format = "email"
		case rule == "url":
			schema.Format = "uri"
		}
	}
}

// Build 构建文档对象。
func (b *Builder) Build() Document {
	return b.doc
}

// Handler 返回 Swagger JSON 处理器。
func (b *Builder) Handler() gin.HandlerFunc {
	doc := b.Build()
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, doc)
	}
}

// UI 返回 Swagger UI 页面处理器。
func (b *Builder) UI(specURL string) gin.HandlerFunc {
	html := swaggerUIHTML(specURL)
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// Redoc 返回 Redoc 页面处理器。
func (b *Builder) Redoc(specURL string) gin.HandlerFunc {
	html := redocHTML(specURL)
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

func ptrInt(v int) *int             { return &v }
func ptrFloat64(v float64) *float64 { return &v }

// swaggerUIHTML 返回 Swagger UI 页面。
func swaggerUIHTML(specURL string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>接口文档</title>
    <link rel="stylesheet" type="text/css" href="https://cdn.bootcdn.net/ajax/libs/swagger-ui/5.11.0/swagger-ui.min.css">
    <style>
        html, body { margin: 0; padding: 0; }
        .swagger-ui .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.bootcdn.net/ajax/libs/swagger-ui/5.11.0/swagger-ui-bundle.min.js"></script>
    <script src="https://cdn.bootcdn.net/ajax/libs/swagger-ui/5.11.0/swagger-ui-standalone-preset.min.js"></script>
    <script>
    function replaceText(root) {
        if (!root) return
        const textMap = new Map([
            ['Authorize', '认证'],
            ['Authorized', '已认证'],
            ['Available authorizations', '可用认证方式'],
            ['Logout', '退出认证'],
            ['Try it out', '开始调试'],
            ['Execute', '执行'],
            ['Clear', '清空'],
            ['Cancel', '取消'],
            ['Responses', '响应'],
            ['Response content type', '响应内容类型'],
            ['Response body', '响应体'],
            ['Request body', '请求体'],
            ['Parameters', '参数'],
            ['No parameters', '无参数'],
            ['No operations defined in spec!', '文档中未定义任何接口'],
            ['Schemas', '数据模型'],
            ['Model', '模型'],
            ['Example Value', '示例值'],
            ['Example', '示例'],
            ['Value', '值'],
            ['Description', '说明'],
            ['Details', '详情'],
            ['Servers', '服务地址'],
            ['Server', '服务地址'],
            ['Filter by tag', '按标签筛选'],
            ['Filter', '筛选'],
            ['Search', '搜索'],
            ['Download', '下载'],
            ['Expand operation', '展开接口'],
            ['Collapse operation', '收起接口']
        ])

        const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
        const textNodes = []
        while (walker.nextNode()) {
            textNodes.push(walker.currentNode)
        }

        textNodes.forEach(function(node) {
            const original = node.nodeValue && node.nodeValue.trim()
            if (!original) return
            if (textMap.has(original)) {
                node.nodeValue = node.nodeValue.replace(original, textMap.get(original))
            }
        })

        root.querySelectorAll('input[placeholder]').forEach(function(el) {
            const placeholderMap = {
                'Filter': '筛选',
                'Search': '搜索'
            }
            if (placeholderMap[el.placeholder]) {
                el.placeholder = placeholderMap[el.placeholder]
            }
        })
    }

    function localizeSwaggerUI() {
        document.title = '接口文档'
        replaceText(document.body)
    }

    window.onload = function() {
        SwaggerUIBundle({
            url: "` + specURL + `",
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            layout: "StandaloneLayout"
        })

        localizeSwaggerUI()

        const observer = new MutationObserver(function() {
            localizeSwaggerUI()
        })
        observer.observe(document.body, {
            childList: true,
            subtree: true
        })
    }
    </script>
</body>
</html>`
}

// redocHTML 返回 Redoc 页面。
func redocHTML(specURL string) string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>API 文档</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { margin: 0; padding: 0; }
    </style>
</head>
<body>
    <div id="redoc-container"></div>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
    <script>
        Redoc.init('` + specURL + `', {
            scrollYOffset: 50,
            hideDownloadButton: false,
            expandResponses: '200,201',
            requiredPropsFirst: true,
            sortPropsAlphabetically: true,
            pathInMiddlePanel: true,
            hideLoading: false,
            nativeScrollbars: true,
            theme: {
                colors: {
                    primary: {
                        main: '#1890ff'
                    }
                },
                typography: {
                    fontSize: '15px',
                    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
                    headings: {
                        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif'
                    }
                },
                sidebar: {
                    width: '260px'
                }
            }
        }, document.getElementById('redoc-container'))
    </script>
</body>
</html>`
}

// OKResponse 创建 200 响应定义。
func OKResponse(description string, schema *Schema) Response {
	return Response{
		Description: description,
		Content: map[string]Content{
			"application/json": {Schema: schema},
		},
	}
}

// ErrorResponse 创建错误响应定义。
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

// JSONBody 创建 JSON 请求体定义。
func JSONBody(schema *Schema, required bool) *RequestBody {
	return &RequestBody{
		Required: required,
		Content: map[string]Content{
			"application/json": {Schema: schema},
		},
	}
}

// QueryParam 创建查询参数。
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

// PathParam 创建路径参数。
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

// HeaderParam 创建请求头参数。
func HeaderParam(name, description string, required bool) Parameter {
	return Parameter{
		Name:        name,
		In:          "header",
		Description: description,
		Required:    required,
		Schema:      &Schema{Type: "string"},
	}
}

// RouteInfo 表示路由信息。
type RouteInfo struct {
	Method      string
	Path        string
	Handler     string
	Description string
	Tags        []string
}

// AutoRegister 自动注册 Gin 路由到文档。
func (b *Builder) AutoRegister(r *gin.Engine) {
	routes := r.Routes()
	for _, route := range routes {
		if strings.HasPrefix(route.Path, "/swagger") {
			continue
		}

		method := strings.ToLower(route.Method)
		path := route.Path
		params := extractPathParams(path)

		item := PathItem{
			Summary: route.Handler,
			Tags:    []string{"default"},
			Responses: map[string]Response{
				"200": OKResponse("成功", nil),
			},
		}

		for _, p := range params {
			item.Parameters = append(item.Parameters, PathParam(p, p+" 参数", nil))
		}

		if b.doc.Paths[path] == nil {
			b.doc.Paths[path] = make(map[string]PathItem)
		}
		b.doc.Paths[path][method] = item
	}
}

// extractPathParams 提取路径参数。
func extractPathParams(path string) []string {
	re := regexp.MustCompile(`:([^/]+)`)
	matches := re.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, m := range matches {
		params = append(params, m[1])
	}
	return params
}

var defaultBuilder *Builder

// Init 初始化默认 Swagger 构建器。
func Init(title, version string) *Builder {
	defaultBuilder = NewBuilder().Title(title).Version(version)
	return defaultBuilder
}

// InitWithConfig 使用配置初始化默认 Swagger 构建器。
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

// Default 返回默认构建器。
func Default() *Builder {
	if defaultBuilder == nil {
		defaultBuilder = NewBuilder()
	}
	return defaultBuilder
}

// Register 注册 Swagger 路由。
//
// 默认会注册：
// - GET /swagger.json
// - GET /swagger/ui
// - GET /swagger/index.html
//
// 注意：Register 不会自动收集业务路由；如果需要自动注册，请使用 Setup。
func Register(r *gin.Engine) *Builder {
	b := Default()

	r.GET("/swagger.json", b.Handler())
	r.GET("/swagger/ui", b.UI("/swagger.json"))
	r.GET("/swagger/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/ui")
	})

	return b
}

// Setup 快速设置 Swagger，并自动收集已有路由。
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

// Marshal 将文档对象转换为 JSON。
func Marshal(doc Document) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}
