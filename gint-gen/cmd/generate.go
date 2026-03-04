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

	"github.com/ink-yht-code/gint-gen/generator"
	"github.com/ink-yht-code/gint-gen/template"
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

	// types.go
	typesContent, err := generator.ExecuteTemplate(template.TypesTmpl, data)
	if err != nil {
		return err
	}
	if err := generator.GenerateFile(filepath.Join(name, "internal", "types", "types.go"), typesContent); err != nil {
		return err
	}

	// handler.go
	handlerContent, err := generator.ExecuteTemplate(template.HTTPTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "internal", "web", "handler.go"), handlerContent)
}

func generateServerFile(name string) error {
	data := generator.ServiceData{
		Name:      name,
		NameUpper: strings.Title(name),
	}
	content, err := generator.ExecuteTemplate(template.ServerTmpl, data)
	if err != nil {
		return err
	}
	return generator.GenerateFile(filepath.Join(name, "internal", "server", name+".go"), content)
}
