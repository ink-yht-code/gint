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

package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TypeDef 类型定义
type TypeDef struct {
	Name   string
	Fields []Field
}

// Field 字段
type Field struct {
	Name     string
	Type     string
	Tag      string
	Validate string
}

// Route 路由
type Route struct {
	Method   string
	Path     string
	Handler  string
	Request  string
	Response string
	Private  bool
}

// Service 服务定义
type Service struct {
	Name   string
	Prefix string
	Routes []Route
}

// API API 定义
type API struct {
	Syntax   string
	Types    []TypeDef
	Services []Service
}

// Parser .gint 文件解析器
type Parser struct{}

// NewParser 创建解析器
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile 解析文件
func (p *Parser) ParseFile(path string) (*API, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.Parse(string(data))
}

// Parse 解析内容
func (p *Parser) Parse(content string) (*API, error) {
	api := &API{
		Types:    make([]TypeDef, 0),
		Services: make([]Service, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentType *TypeDef
	var currentService *Service
	var inType bool
	var inService bool
	var inServerBlock bool
	var nextRoutePrivate bool
	var inServerBlockV2 bool
	var inPublicBlock bool
	var inPrivateBlock bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// syntax
		if strings.HasPrefix(line, "syntax") {
			re := regexp.MustCompile(`syntax\s*=\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				api.Syntax = matches[1]
			}
			continue
		}

		// server v2 block start: server {
		if strings.HasPrefix(line, "server") && strings.Contains(line, "{") {
			inServerBlockV2 = true
			if currentService == nil {
				currentService = &Service{}
			}
			continue
		}

		// server v2 block end
		if inServerBlockV2 && line == "}" {
			inServerBlockV2 = false
			continue
		}

		// server v2 block content
		if inServerBlockV2 {
			// prefix "/api/v1"
			re := regexp.MustCompile(`^prefix\s+"([^"]+)"$`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentService.Prefix = matches[1]
			}
			continue
		}

		// type 定义开始
		if strings.HasPrefix(line, "type ") {
			inType = true
			name := strings.TrimPrefix(line, "type ")
			name = strings.TrimSuffix(name, "{")
			name = strings.TrimSpace(name)
			currentType = &TypeDef{Name: name, Fields: make([]Field, 0)}
			continue
		}

		// type 结束
		if inType && line == "}" {
			inType = false
			if currentType != nil {
				api.Types = append(api.Types, *currentType)
				currentType = nil
			}
			continue
		}

		// type 字段
		if inType && currentType != nil {
			field := p.parseField(line)
			if field.Name != "" {
				currentType.Fields = append(currentType.Fields, field)
			}
			continue
		}

		// @server 块开始
		if strings.HasPrefix(line, "@server(") {
			inServerBlock = true
			if currentService == nil {
				currentService = &Service{}
			}
			// 解析 prefix
			re := regexp.MustCompile(`prefix:\s*([^\s\)]+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentService.Prefix = matches[1]
			}
			continue
		}

		// @server 块结束
		if inServerBlock && line == ")" {
			inServerBlock = false
			continue
		}

		// service 定义开始
		if strings.HasPrefix(line, "service ") {
			inService = true
			nextRoutePrivate = false
			inPublicBlock = false
			inPrivateBlock = false
			name := strings.TrimPrefix(line, "service ")
			name = strings.TrimSuffix(name, "{")
			name = strings.TrimSpace(name)
			if currentService == nil {
				currentService = &Service{Name: name}
			} else {
				currentService.Name = name
			}
			continue
		}

		// service 结束
		if inService && line == "}" {
			inService = false
			nextRoutePrivate = false
			inPublicBlock = false
			inPrivateBlock = false
			if currentService != nil {
				api.Services = append(api.Services, *currentService)
				currentService = nil
			}
			continue
		}

		// v2 public/private blocks
		if inService {
			if strings.HasPrefix(line, "public") && strings.Contains(line, "{") {
				inPublicBlock = true
				inPrivateBlock = false
				nextRoutePrivate = false
				continue
			}
			if strings.HasPrefix(line, "private") && strings.Contains(line, "{") {
				inPrivateBlock = true
				inPublicBlock = false
				nextRoutePrivate = true
				continue
			}
			if (inPublicBlock || inPrivateBlock) && line == "}" {
				inPublicBlock = false
				inPrivateBlock = false
				nextRoutePrivate = false
				continue
			}
		}

		// 路由选项：@private
		if inService && strings.HasPrefix(line, "@private") {
			nextRoutePrivate = true
			continue
		}

		// 路由定义
		if inService && currentService != nil {
			route := p.parseRoute(line)
			if route.Method != "" {
				// v2 public/private blocks take precedence
				if inPrivateBlock {
					route.Private = true
				} else if inPublicBlock {
					route.Private = false
				} else {
					route.Private = nextRoutePrivate
				}
				currentService.Routes = append(currentService.Routes, route)
				nextRoutePrivate = false
			}
		}
	}

	return api, nil
}

// parseField 解析字段
func (p *Parser) parseField(line string) Field {
	// 格式: Name string `json:"name" validate:"required"`
	// 使用 raw string 避免转义问题
	re := regexp.MustCompile(`(\w+)\s+(\w+)(?:\s+\x60([^\x60]*)\x60)?`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 3 {
		field := Field{
			Name: matches[1],
			Type: matches[2],
		}
		if len(matches) > 3 {
			field.Tag = matches[3]
			// 提取 validate
			validateRe := regexp.MustCompile(`validate:"([^"]+)"`)
			validateMatches := validateRe.FindStringSubmatch(matches[3])
			if len(validateMatches) > 1 {
				field.Validate = validateMatches[1]
			}
		}
		return field
	}
	return Field{}
}

// parseRoute 解析路由
func (p *Parser) parseRoute(line string) Route {
	// 支持两种格式:
	// 格式1: POST "/register" Register(RegisterReq) -> HelloResp
	// 格式2: POST /users (CreateUserReq) returns (CreateUserResp)
	route := Route{}

	// @handler
	if strings.HasPrefix(line, "@handler ") {
		route.Handler = strings.TrimPrefix(line, "@handler ")
		return route
	}

	// HTTP 方法
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		if strings.HasPrefix(line, method+" ") {
			route.Method = method
			rest := strings.TrimPrefix(line, method+" ")

			// 格式1: "/register" Register(RegisterReq) -> HelloResp
			// 检查是否包含 -> 箭头
			if strings.Contains(rest, "->") {
				// 解析格式1
				parts := strings.SplitN(rest, "->", 2)
				leftPart := strings.TrimSpace(parts[0])
				rightPart := strings.TrimSpace(parts[1])

				// 解析响应类型
				route.Response = strings.TrimSpace(rightPart)

				// 解析左边: "/register" Register(RegisterReq)
				// 提取路径
				pathRe := regexp.MustCompile(`^"([^"]+)"\s*`)
				pathMatches := pathRe.FindStringSubmatch(leftPart)
				if len(pathMatches) > 1 {
					route.Path = pathMatches[1]
					leftPart = strings.TrimPrefix(leftPart, pathMatches[0])
				} else {
					// 路径没有引号
					pathParts := strings.SplitN(leftPart, " ", 2)
					route.Path = strings.TrimSpace(pathParts[0])
					if len(pathParts) > 1 {
						leftPart = pathParts[1]
					}
				}

				// 解析 Handler(Req)
				handlerRe := regexp.MustCompile(`(\w+)\((\w*)\)`)
				handlerMatches := handlerRe.FindStringSubmatch(leftPart)
				if len(handlerMatches) > 2 {
					route.Handler = handlerMatches[1]
					route.Request = handlerMatches[2]
				}
				return route
			}

			// 格式2: /users (CreateUserReq) returns (CreateUserResp)
			parts := strings.SplitN(rest, "(", 2)
			route.Path = strings.TrimSpace(parts[0])
			// 移除路径中的引号
			route.Path = strings.Trim(route.Path, `"`)

			if len(parts) > 1 {
				// 请求类型
				reqRe := regexp.MustCompile(`\((\w+)\)`)
				reqMatches := reqRe.FindStringSubmatch(parts[1])
				if len(reqMatches) > 1 {
					route.Request = reqMatches[1]
				}

				// 响应类型
				respRe := regexp.MustCompile(`returns\s*\((\w+)\)`)
				respMatches := respRe.FindStringSubmatch(parts[1])
				if len(respMatches) > 1 {
					route.Response = respMatches[1]
				}
			}
			return route
		}
	}

	return Route{}
}

// GoType Go 类型映射
func GoType(gintType string) string {
	switch gintType {
	case "int":
		return "int"
	case "int64":
		return "int64"
	case "int32":
		return "int32"
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "float64":
		return "float64"
	case "[]string":
		return "[]string"
	case "[]int":
		return "[]int"
	case "[]int64":
		return "[]int64"
	default:
		return gintType
	}
}

// Validate 验证 API 定义
func (a *API) Validate() error {
	if len(a.Types) == 0 {
		return fmt.Errorf("no types defined")
	}
	if len(a.Services) == 0 {
		return fmt.Errorf("no services defined")
	}
	return nil
}
