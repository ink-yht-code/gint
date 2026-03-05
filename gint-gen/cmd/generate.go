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
	"path/filepath"
	"strings"

	"github.com/ink-yht-code/gint/gint-gen/generator"
	"github.com/ink-yht-code/gint/gint-gen/template"
)

func generateGintFile(name string) error {
	data := generator.ServiceData{
		Name:      name,
		NameUpper: strings.Title(name),
	}
	content, err := generator.ExecuteTemplate(template.GintTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, name+".gint"), content)
}

func generateConfigFile(name string, serviceID int, hasHTTP, hasRPC bool) error {
	data := generator.ServiceData{
		Name:      name,
		ServiceID: serviceID,
		HasHTTP:   hasHTTP,
		HasRPC:    hasRPC,
	}
	content, err := generator.ExecuteTemplate(template.ConfigYamlTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "configs", name+".yaml"), content)
}

func generateMainFile(name string) error {
	data := generator.ServiceData{
		Name: name,
	}
	content, err := generator.ExecuteTemplate(template.MainTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "cmd", "main.go"), content)
}

func generateConfigGo(name string) error {
	content := template.ConfigGoTmpl
	return generator.GenerateFile(filepath.Join(name, "internal", "config", "config.go"), content)
}

func generateCodesFile(name string, serviceID int) error {
	data := generator.ServiceData{
		ServiceID: serviceID,
	}
	content, err := generator.ExecuteTemplate(template.CodesTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "internal", "domain", "errs", "codes.go"), content)
}

func generateErrorFile(name string) error {
	content := template.ErrorTmpl
	return generator.GenerateFile(filepath.Join(name, "internal", "domain", "errs", "error.go"), content)
}

func generateWiringFiles(name string, hasHTTP, hasRPC bool) error {
	data := generator.ServiceData{
		Name:    name,
		HasHTTP: hasHTTP,
		HasRPC:  hasRPC,
	}
	content, err := generator.ExecuteTemplate(template.WiringTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "internal", "wiring", "wiring.go"), content)
}

func generateHTTPFiles(name string) error {
	data := generator.ServiceData{
		Name:      name,
		NameUpper: strings.Title(name),
	}

	// types_gen.go (generated)
	typesContent, err := generator.ExecuteTemplate(template.TypesTmpl, data)
	if err != nil {
		return err
	}
	if err := generator.GenerateFile(filepath.Join(name, "internal", "types", "types_gen.go"), typesContent); err != nil {
		return err
	}

	// handler_gen.go (generated, overwriteable)
	handlerGenContent, err := generator.ExecuteTemplate(template.HTTPHandlerGenTmpl, data)
	if err != nil {
		return err
	}
	if err := generator.GenerateFile(filepath.Join(name, "internal", "web", "handler_gen.go"), handlerGenContent); err != nil {
		return err
	}

	// <service>_handlers.go (user-editable, create once)
	handlersContent, err := generator.ExecuteTemplate(template.HTTPHandlerImplTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "internal", "web", name+"_handlers.go"), handlersContent)
}

func generateServerFile(name string) error {
	data := generator.ServiceData{
		Name:      name,
		NameUpper: strings.Title(name),
	}
	content, err := generator.ExecuteTemplate(template.ServiceTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "internal", "service", name+".go"), content)
}

func generateRepositoryFiles(name string) error {
	data := generator.ServiceData{
		Name:      name,
		NameUpper: strings.Title(name),
	}

	// entity
	entityContent, err := generator.ExecuteTemplate(template.EntityTmpl, data)
	if err != nil {
		return err
	}
	if err := generator.GenerateFile(filepath.Join(name, "internal", "domain", "entity", name+".go"), entityContent); err != nil {
		return err
	}

	// repository port
	portContent, err := generator.ExecuteTemplate(template.RepositoryPortTmpl, data)
	if err != nil {
		return err
	}
	if err := generator.GenerateFile(filepath.Join(name, "internal", "domain", "port", "repository.go"), portContent); err != nil {
		return err
	}

	// DAO interface
	daoContent, err := generator.ExecuteTemplate(template.DAOTmpl, data)
	if err != nil {
		return err
	}
	if err := generator.GenerateFile(filepath.Join(name, "internal", "repository", "dao", name+".go"), daoContent); err != nil {
		return err
	}

	// repository implementation
	repoContent, err := generator.ExecuteTemplate(template.RepositoryImplTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "internal", "repository", name+".go"), repoContent)
}

func generateGoMod(name string) error {
	data := generator.ServiceData{
		Name: name,
	}
	content, err := generator.ExecuteTemplate(template.GoModTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "go.mod"), content)
}
