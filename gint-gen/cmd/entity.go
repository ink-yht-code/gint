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

var entityCmd = &cobra.Command{
	Use:   "entity <name>",
	Short: "生成实体四层代码",
	Long: `生成实体的 entity、port、dao、repository 四层代码。

示例:
  gint-gen entity user
  gint-gen entity order --fields "name:string,price:int64"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		fields, _ := cmd.Flags().GetString("fields")
		if err := generateEntity(name, fields); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Entity '%s' generated successfully.\n", name)
	},
}

func init() {
	rootCmd.AddCommand(entityCmd)
	entityCmd.Flags().String("fields", "", "字段定义: name:string,age:int")
}

// EntityField 实体字段
type EntityField struct {
	Name string
	Type string
}

func generateEntity(name string, fieldsStr string) error {
	// 解析字段
	fields := parseFields(fieldsStr)

	// 获取模块名
	moduleName := getModuleNameFromGoMod()
	if moduleName == "" {
		moduleName = name // fallback
	}

	// 检查是否在服务目录内
	inServiceDir := false
	if _, err := os.Stat("internal/domain/entity"); err == nil {
		inServiceDir = true
	}

	// 生成 entity
	if err := genEntityFile(name, fields, inServiceDir); err != nil {
		return err
	}

	// 生成 port (repository interface)
	if err := genPortFile(name, moduleName, inServiceDir); err != nil {
		return err
	}

	// 生成 dao
	if err := genDAOFile(name, moduleName, inServiceDir); err != nil {
		return err
	}

	// 生成 repository
	if err := genRepositoryFile(name, moduleName, inServiceDir); err != nil {
		return err
	}

	return nil
}

func parseFields(fieldsStr string) []EntityField {
	if fieldsStr == "" {
		return []EntityField{}
	}
	var fields []EntityField
	for _, f := range strings.Split(fieldsStr, ",") {
		parts := strings.Split(strings.TrimSpace(f), ":")
		if len(parts) == 2 {
			fields = append(fields, EntityField{Name: parts[0], Type: parts[1]})
		}
	}
	return fields
}

func genEntityFile(name string, fields []EntityField, inServiceDir bool) error {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("// %s %s 实体\n", strings.Title(name), name))
	buf.WriteString(fmt.Sprintf("type %s struct {\n", strings.Title(name)))
	buf.WriteString("\tID        int64 `json:\"id\"`\n")
	for _, f := range fields {
		goType := goTypeFromName(f.Type)
		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", strings.Title(f.Name), goType, f.Name))
	}
	buf.WriteString("\tCreatedAt int64 `json:\"created_at\"`\n")
	buf.WriteString("\tUpdatedAt int64 `json:\"updated_at\"`\n")
	buf.WriteString("}\n")

	entityPath := filepath.Join("internal", "domain", "entity", strings.ToLower(name)+".go")
	if !inServiceDir {
		entityPath = filepath.Join(name, "internal", "domain", "entity", strings.ToLower(name)+".go")
	}

	return generator.GenerateFile(entityPath, "package entity\n\n"+buf.String())
}

func genPortFile(name, moduleName string, inServiceDir bool) error {
	nameTitle := strings.Title(name)
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("// %sRepository %s 仓储接口\n", nameTitle, name))
	buf.WriteString("type " + nameTitle + "Repository interface {\n")
	buf.WriteString(fmt.Sprintf("\tCreate(ctx interface{}, entity *entity.%s) error\n", nameTitle))
	buf.WriteString(fmt.Sprintf("\tFindByID(ctx interface{}, id int64) (*entity.%s, error)\n", nameTitle))
	buf.WriteString(fmt.Sprintf("\tUpdate(ctx interface{}, entity *entity.%s) error\n", nameTitle))
	buf.WriteString("\tDelete(ctx interface{}, id int64) error\n")
	buf.WriteString("}\n")

	portPath := filepath.Join("internal", "domain", "port", strings.ToLower(name)+"_repository.go")
	if !inServiceDir {
		portPath = filepath.Join(name, "internal", "domain", "port", strings.ToLower(name)+"_repository.go")
	}

	imports := fmt.Sprintf("import (\n\t\"%s/internal/domain/entity\"\n)\n\n", moduleName)
	return generator.GenerateFile(portPath, "package port\n\n"+imports+buf.String())
}

func genDAOFile(name, moduleName string, inServiceDir bool) error {
	nameTitle := strings.Title(name)

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("// %sDAO %s DAO 接口\n", nameTitle, name))
	buf.WriteString("type " + nameTitle + "DAO interface {\n")
	buf.WriteString(fmt.Sprintf("\tCreate(ctx interface{}, entity *entity.%s) error\n", nameTitle))
	buf.WriteString(fmt.Sprintf("\tFindByID(ctx interface{}, id int64) (*entity.%s, error)\n", nameTitle))
	buf.WriteString(fmt.Sprintf("\tUpdate(ctx interface{}, entity *entity.%s) error\n", nameTitle))
	buf.WriteString("\tDelete(ctx interface{}, id int64) error\n")
	buf.WriteString("}\n\n")

	// 内存实现
	buf.WriteString(fmt.Sprintf("// %sDAOImpl 内存实现\n", nameTitle))
	buf.WriteString(fmt.Sprintf("type %sDAOImpl struct {\n", nameTitle))
	buf.WriteString("\tmu     sync.RWMutex\n")
	buf.WriteString(fmt.Sprintf("\tdata   map[int64]*entity.%s\n", nameTitle))
	buf.WriteString("\tnextID int64\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("// New%sDAO 创建 DAO\n", nameTitle))
	buf.WriteString(fmt.Sprintf("func New%sDAO() *%sDAOImpl {\n", nameTitle, nameTitle))
	buf.WriteString("\treturn &" + nameTitle + "DAOImpl{\n")
	buf.WriteString("\t\tdata:   make(map[int64]*entity." + nameTitle + "),\n")
	buf.WriteString("\t\tnextID: 1,\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	// CRUD 方法
	buf.WriteString("func (d *" + nameTitle + "DAOImpl) Create(ctx interface{}, entity *" + nameTitle + ") error {\n")
	buf.WriteString("\td.mu.Lock()\n")
	buf.WriteString("\tdefer d.mu.Unlock()\n")
	buf.WriteString("\tentity.ID = d.nextID\n")
	buf.WriteString("\td.nextID++\n")
	buf.WriteString("\td.data[entity.ID] = entity\n")
	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	buf.WriteString("func (d *" + nameTitle + "DAOImpl) FindByID(ctx interface{}, id int64) (*" + nameTitle + ", error) {\n")
	buf.WriteString("\td.mu.RLock()\n")
	buf.WriteString("\tdefer d.mu.RUnlock()\n")
	buf.WriteString("\treturn d.data[id], nil\n")
	buf.WriteString("}\n\n")

	buf.WriteString("func (d *" + nameTitle + "DAOImpl) Update(ctx interface{}, entity *" + nameTitle + ") error {\n")
	buf.WriteString("\td.mu.Lock()\n")
	buf.WriteString("\tdefer d.mu.Unlock()\n")
	buf.WriteString("\td.data[entity.ID] = entity\n")
	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	buf.WriteString("func (d *" + nameTitle + "DAOImpl) Delete(ctx interface{}, id int64) error {\n")
	buf.WriteString("\td.mu.Lock()\n")
	buf.WriteString("\tdefer d.mu.Unlock()\n")
	buf.WriteString("\tdelete(d.data, id)\n")
	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n")

	daoPath := filepath.Join("internal", "repository", "dao", strings.ToLower(name)+".go")
	if !inServiceDir {
		daoPath = filepath.Join(name, "internal", "repository", "dao", strings.ToLower(name)+".go")
	}

	imports := fmt.Sprintf("import (\n\t\"sync\"\n\t\"%s/internal/domain/entity\"\n)\n\n", moduleName)
	return generator.GenerateFile(daoPath, "package dao\n\n"+imports+buf.String())
}

func genRepositoryFile(name, moduleName string, inServiceDir bool) error {
	nameTitle := strings.Title(name)

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("// %sRepository %s 仓储实现\n", nameTitle, name))
	buf.WriteString(fmt.Sprintf("type %sRepository struct {\n", nameTitle))
	buf.WriteString(fmt.Sprintf("\tdao *dao.%sDAOImpl\n", nameTitle))
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("// New%sRepository 创建仓储\n", nameTitle))
	buf.WriteString(fmt.Sprintf("func New%sRepository(dao *dao.%sDAOImpl) port.%sRepository {\n", nameTitle, nameTitle, nameTitle))
	buf.WriteString(fmt.Sprintf("\treturn &%sRepository{dao: dao}\n", nameTitle))
	buf.WriteString("}\n\n")

	// CRUD 方法
	buf.WriteString("func (r *" + nameTitle + "Repository) Create(ctx interface{}, entity *entity." + nameTitle + ") error {\n")
	buf.WriteString("\treturn r.dao.Create(ctx, entity)\n")
	buf.WriteString("}\n\n")

	buf.WriteString("func (r *" + nameTitle + "Repository) FindByID(ctx interface{}, id int64) (*entity." + nameTitle + ", error) {\n")
	buf.WriteString("\treturn r.dao.FindByID(ctx, id)\n")
	buf.WriteString("}\n\n")

	buf.WriteString("func (r *" + nameTitle + "Repository) Update(ctx interface{}, entity *entity." + nameTitle + ") error {\n")
	buf.WriteString("\treturn r.dao.Update(ctx, entity)\n")
	buf.WriteString("}\n\n")

	buf.WriteString("func (r *" + nameTitle + "Repository) Delete(ctx interface{}, id int64) error {\n")
	buf.WriteString("\treturn r.dao.Delete(ctx, id)\n")
	buf.WriteString("}\n")

	repoPath := filepath.Join("internal", "repository", strings.ToLower(name)+".go")
	if !inServiceDir {
		repoPath = filepath.Join(name, "internal", "repository", strings.ToLower(name)+".go")
	}

	imports := fmt.Sprintf("import (\n\t\"%s/internal/domain/entity\"\n\t\"%s/internal/domain/port\"\n\t\"%s/internal/repository/dao\"\n)\n\n", moduleName, moduleName, moduleName)
	return generator.GenerateFile(repoPath, "package repository\n\n"+imports+buf.String())
}

func goTypeFromName(name string) string {
	switch name {
	case "string":
		return "string"
	case "int":
		return "int"
	case "int64":
		return "int64"
	case "int32":
		return "int32"
	case "float", "float64":
		return "float64"
	case "bool":
		return "bool"
	default:
		return name
	}
}
