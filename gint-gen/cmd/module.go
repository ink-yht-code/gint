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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ink-yht-code/gint/gint-gen/generator"
	"github.com/spf13/cobra"
)

var moduleCmd = &cobra.Command{
	Use:   "module <name>",
	Short: "生成完整模块",
	Long: `生成完整的业务模块，包含 entity、port、dao、repository、service、handler。

内置模块:
  user   - 用户模块（注册、登录、JWT认证）

示例:
  gint-gen module user
  gint-gen module order --fields "name:string,price:int64"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		fields, _ := cmd.Flags().GetString("fields")
		if err := generateModule(name, fields); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Module '%s' generated successfully.\n", name)
	},
}

func init() {
	rootCmd.AddCommand(moduleCmd)
	moduleCmd.Flags().String("fields", "", "字段定义: name:string,age:int")
}

func generateModule(name string, fieldsStr string) error {
	// 获取模块名
	moduleName := getModuleNameFromGoMod()
	if moduleName == "" {
		moduleName = name
	}

	// 检查是否在服务目录内
	inServiceDir := false
	if _, err := os.Stat("internal/domain/entity"); err == nil {
		inServiceDir = true
	}

	// 生成四层代码
	if err := generateEntity(name, fieldsStr); err != nil {
		return err
	}

	// 生成 service
	if err := genModuleService(name, moduleName, inServiceDir); err != nil {
		return err
	}

	// 如果是 user 模块，生成额外的认证代码
	if name == "user" {
		if err := genUserAuthCode(moduleName, inServiceDir); err != nil {
			return err
		}
	}

	return nil
}

func genModuleService(name, moduleName string, inServiceDir bool) error {
	nameTitle := strings.Title(name)

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("// %sService %s 服务\n", nameTitle, name))
	buf.WriteString(fmt.Sprintf("type %sService struct {\n", nameTitle))
	buf.WriteString(fmt.Sprintf("\trepo port.%sRepository\n", nameTitle))
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("// New%sService 创建服务\n", nameTitle))
	buf.WriteString(fmt.Sprintf("func New%sService(repo port.%sRepository) *%sService {\n", nameTitle, nameTitle, nameTitle))
	buf.WriteString(fmt.Sprintf("\treturn &%sService{repo: repo}\n", nameTitle))
	buf.WriteString("}\n\n")

	// CRUD 方法
	buf.WriteString(fmt.Sprintf("// Create 创建%s\n", name))
	buf.WriteString(fmt.Sprintf("func (s *%sService) Create(ctx interface{}, entity *entity.%s) error {\n", nameTitle, nameTitle))
	buf.WriteString("\treturn s.repo.Create(ctx, entity)\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("// GetByID 根据ID获取%s\n", name))
	buf.WriteString(fmt.Sprintf("func (s *%sService) GetByID(ctx interface{}, id int64) (*entity.%s, error) {\n", nameTitle, nameTitle))
	buf.WriteString("\treturn s.repo.FindByID(ctx, id)\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("// Update 更新%s\n", name))
	buf.WriteString(fmt.Sprintf("func (s *%sService) Update(ctx interface{}, entity *entity.%s) error {\n", nameTitle, nameTitle))
	buf.WriteString("\treturn s.repo.Update(ctx, entity)\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("// Delete 删除%s\n", name))
	buf.WriteString(fmt.Sprintf("func (s *%sService) Delete(ctx interface{}, id int64) error {\n", nameTitle, nameTitle))
	buf.WriteString("\treturn s.repo.Delete(ctx, id)\n")
	buf.WriteString("}\n")

	svcPath := filepath.Join("internal", "service", strings.ToLower(name)+".go")
	if !inServiceDir {
		svcPath = filepath.Join(name, "internal", "service", strings.ToLower(name)+".go")
	}

	imports := fmt.Sprintf("import (\n\t\"%s/internal/domain/entity\"\n\t\"%s/internal/domain/port\"\n)\n\n", moduleName, moduleName)
	return generator.GenerateFile(svcPath, "package service\n\n"+imports+buf.String())
}

func genUserAuthCode(moduleName string, inServiceDir bool) error {
	// 生成 .gint 文件
	if err := genUserGintFile(inServiceDir); err != nil {
		return err
	}

	// 生成 auth service 扩展
	if err := genUserAuthService(moduleName, inServiceDir); err != nil {
		return err
	}

	return nil
}

func genUserGintFile(inServiceDir bool) error {
	content := `syntax = "v1"

server {
    prefix "/api/v1"
}

// RegisterReq 注册请求
type RegisterReq {
    Username string ` + "`" + `json:"username" validate:"required"` + "`" + `
    Password string ` + "`" + `json:"password" validate:"required,min=6"` + "`" + `
    Email    string ` + "`" + `json:"email" validate:"required,email"` + "`" + `
}

// LoginReq 登录请求
type LoginReq {
    Username string ` + "`" + `json:"username" validate:"required"` + "`" + `
    Password string ` + "`" + `json:"password" validate:"required"` + "`" + `
}

// LoginResp 登录响应
type LoginResp {
    AccessToken  string ` + "`" + `json:"access_token"` + "`" + `
    RefreshToken string ` + "`" + `json:"refresh_token"` + "`" + `
}

service mytest {
    public {
        POST "/register" Register(RegisterReq) -> LoginResp
        POST "/login" Login(LoginReq) -> LoginResp
    }
}
`

	gintPath := "mytest.gint"
	if !inServiceDir {
		gintPath = filepath.Join("mytest", "mytest.gint")
	}

	return generator.GenerateFile(gintPath, content)
}

func genUserAuthService(moduleName string, inServiceDir bool) error {
	var buf strings.Builder
	buf.WriteString(`// Register 用户注册
func (s *UserService) Register(ctx interface{}, req *types.RegisterReq) (*types.LoginResp, error) {
	// 检查用户名是否已存在
	existing, _ := s.repo.FindByUsername(ctx, req.Username)
	if existing != nil {
		return nil, fmt.Errorf("username already exists")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &entity.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 生成 JWT token
	tokenPair, err := s.jwtManager.GenerateTokenPair(jwt.Claims{UserId: user.ID})
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

// Login 用户登录
func (s *UserService) Login(ctx interface{}, req *types.LoginReq) (*types.LoginResp, error) {
	// 查找用户
	user, err := s.repo.FindByUsername(ctx, req.Username)
	if err != nil || user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// 生成 JWT token
	tokenPair, err := s.jwtManager.GenerateTokenPair(jwt.Claims{UserId: user.ID})
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}
`)

	// 追加到 service 文件
	svcPath := filepath.Join("internal", "service", "user.go")
	if !inServiceDir {
		svcPath = filepath.Join("mytest", "internal", "service", "user.go")
	}

	f, err := os.OpenFile(svcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n" + buf.String())
	return err
}
