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

	"github.com/spf13/cobra"
)

var (
	registryURL string
	verbose     bool
)

var rootCmd = &cobra.Command{
	Use:   "gint-gen",
	Short: "gint 代码生成器",
	Long: `gint-gen 是一个基于 DDD + Clean Architecture 的微服务代码生成器。

支持生成：
  - 服务骨架（HTTP + gRPC 可选）
  - HTTP 接口（从 .gint 文件）
  - gRPC 接口（从 .proto 文件）
  - Repository（从 SQL）`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&registryURL, "registry", "http://127.0.0.1:18080", "Registry 服务地址")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")
}
