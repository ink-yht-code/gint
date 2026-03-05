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

	"github.com/ink-yht-code/gint/gint-gen/registry"
	"github.com/spf13/cobra"
)

var (
	transport string
	daoType   string
	cacheType string
)

var newServiceCmd = &cobra.Command{
	Use:   "service <name>",
	Short: "创建新服务",
	Long: `创建新服务骨架。

示例:
  gint-gen new service user --transport http,rpc
  gint-gen new service order --transport http`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := createService(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Service '%s' created successfully.\n", name)
	},
}

func init() {
	newCmd.AddCommand(newServiceCmd)
	newServiceCmd.Flags().StringVarP(&transport, "transport", "t", "http", "传输协议: http, rpc, http,rpc")
	newServiceCmd.Flags().StringVar(&daoType, "dao", "gorm", "DAO 类型: gorm")
	newServiceCmd.Flags().StringVar(&cacheType, "cache", "redis", "Cache 类型: redis")
}

func createService(name string) error {
	// 获取 ServiceID
	serviceID, err := allocateServiceID(name)
	if err != nil {
		return fmt.Errorf("分配 ServiceID 失败: %w", err)
	}

	// 创建目录结构
	dirs := []string{
		filepath.Join(name, "cmd"),
		filepath.Join(name, "configs"),
		filepath.Join(name, "internal", "config"),
		filepath.Join(name, "internal", "domain", "errs"),
		filepath.Join(name, "internal", "domain", "entity"),
		filepath.Join(name, "internal", "domain", "port"),
		filepath.Join(name, "internal", "domain", "event"),
		filepath.Join(name, "internal", "repository", "dao"),
		filepath.Join(name, "internal", "repository", "cache"),
		filepath.Join(name, "internal", "repository", "outbox"),
		filepath.Join(name, "internal", "types"),
		filepath.Join(name, "internal", "service"),
		filepath.Join(name, "internal", "web"),
		filepath.Join(name, "internal", "wiring"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}
	}

	// 生成文件
	if err := generateServiceFiles(name, serviceID); err != nil {
		return fmt.Errorf("生成文件失败: %w", err)
	}

	return nil
}

func allocateServiceID(name string) (int, error) {
	// 调用 Registry 服务
	client := registry.NewClient(registryURL, "")
	resp, err := client.Allocate(name)
	if err != nil {
		// 如果 Registry 不可用，使用默认 ID
		if verbose {
			fmt.Printf("Warning: registry service unavailable, using default ID: %v\n", err)
		}
		return 101, nil
	}
	return resp.ServiceID, nil
}

func generateServiceFiles(name string, serviceID int) error {
	hasHTTP := strings.Contains(transport, "http")
	hasRPC := strings.Contains(transport, "rpc")

	// 生成 .gint 文件
	if hasHTTP {
		if err := generateGintFile(name); err != nil {
			return err
		}
	}

	// 生成配置文件
	if err := generateConfigFile(name, serviceID, hasHTTP, hasRPC); err != nil {
		return err
	}

	// 生成 main.go
	if err := generateMainFile(name); err != nil {
		return err
	}

	// 生成 config.go
	if err := generateConfigGo(name); err != nil {
		return err
	}

	// 生成错误码
	if err := generateCodesFile(name, serviceID); err != nil {
		return err
	}

	// 生成 BizError
	if err := generateErrorFile(name); err != nil {
		return err
	}

	// 生成 wiring
	if err := generateWiringFiles(name, hasHTTP, hasRPC); err != nil {
		return err
	}

	// 生成 web
	if hasHTTP {
		if err := generateHTTPFiles(name); err != nil {
			return err
		}
	}

	// 生成 service 示例
	if err := generateServerFile(name); err != nil {
		return err
	}

	// 生成 repository 层 (entity, port, dao, repo)
	if err := generateRepositoryFiles(name); err != nil {
		return err
	}

	// 生成 go.mod
	if err := generateGoMod(name); err != nil {
		return err
	}

	return nil
}
