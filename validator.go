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

package gint

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
)

// ValidationRule 校验规则接口（策略模式）
type ValidationRule interface {
	Validate(value any) error
}

// FieldValidator 字段校验器（建造者模式）
type FieldValidator struct {
	fieldName string
	value     any
	rules     []ValidationRule
	errors    []string
}

// NewFieldValidator 创建字段校验器
func NewFieldValidator(fieldName string, value any) *FieldValidator {
	return &FieldValidator{
		fieldName: fieldName,
		value:     value,
		rules:     make([]ValidationRule, 0),
		errors:    make([]string, 0),
	}
}

// AddRule 添加校验规则
func (fv *FieldValidator) AddRule(rule ValidationRule) *FieldValidator {
	fv.rules = append(fv.rules, rule)
	return fv
}

// Validate 执行校验
func (fv *FieldValidator) Validate() []string {
	for _, rule := range fv.rules {
		if err := rule.Validate(fv.value); err != nil {
			fv.errors = append(fv.errors, fmt.Sprintf("%s%s", fv.fieldName, err.Error()))
		}
	}
	return fv.errors
}

// ValidatorBuilder 校验器构建器（建造者模式）
type ValidatorBuilder struct {
	validators []*FieldValidator
	errors     []string
}

// NewValidatorBuilder 创建校验器构建器
func NewValidatorBuilder() *ValidatorBuilder {
	return &ValidatorBuilder{
		validators: make([]*FieldValidator, 0),
		errors:     make([]string, 0),
	}
}

// Field 添加字段校验
func (vb *ValidatorBuilder) Field(fieldName string, value any) *FieldValidator {
	fv := NewFieldValidator(fieldName, value)
	vb.validators = append(vb.validators, fv)
	return fv
}

// Validate 执行所有校验
func (vb *ValidatorBuilder) Validate() *ValidatorBuilder {
	for _, validator := range vb.validators {
		errors := validator.Validate()
		vb.errors = append(vb.errors, errors...)
	}
	return vb
}

// IsValid 检查是否通过校验
func (vb *ValidatorBuilder) IsValid() bool {
	return len(vb.errors) == 0
}

// GetErrors 获取所有错误
func (vb *ValidatorBuilder) GetErrors() []string {
	return vb.errors
}

// GetFirstError 获取第一个错误
func (vb *ValidatorBuilder) GetFirstError() string {
	if len(vb.errors) > 0 {
		return vb.errors[0]
	}
	return ""
}

// GetErrorString 获取错误字符串
func (vb *ValidatorBuilder) GetErrorString() string {
	return strings.Join(vb.errors, "；")
}

// ============ 具体的校验规则实现（策略模式） ============

// RequiredRule 必填规则
type RequiredRule struct{}

func (r *RequiredRule) Validate(value any) error {
	if value == nil {
		return fmt.Errorf("不能为空")
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("不能为空")
		}
	case int, int8, int16, int32, int64:
		// 数字类型不校验
	}

	return nil
}

// Required 必填规则构造函数
func Required() ValidationRule {
	return &RequiredRule{}
}

// MinLengthRule 最小长度规则
type MinLengthRule struct {
	min int
}

func (r *MinLengthRule) Validate(value any) error {
	str, ok := value.(string)
	if !ok {
		return nil
	}

	if utf8.RuneCountInString(str) < r.min {
		return fmt.Errorf("长度不能少于%d个字符", r.min)
	}

	return nil
}

// MinLength 最小长度规则构造函数
func MinLength(min int) ValidationRule {
	return &MinLengthRule{min: min}
}

// MaxLengthRule 最大长度规则
type MaxLengthRule struct {
	max int
}

func (r *MaxLengthRule) Validate(value any) error {
	str, ok := value.(string)
	if !ok {
		return nil
	}

	if utf8.RuneCountInString(str) > r.max {
		return fmt.Errorf("长度不能超过%d个字符", r.max)
	}

	return nil
}

// MaxLength 最大长度规则构造函数
func MaxLength(max int) ValidationRule {
	return &MaxLengthRule{max: max}
}

// LengthRangeRule 长度范围规则
type LengthRangeRule struct {
	min int
	max int
}

func (r *LengthRangeRule) Validate(value any) error {
	str, ok := value.(string)
	if !ok {
		return nil
	}

	length := utf8.RuneCountInString(str)
	if length < r.min || length > r.max {
		return fmt.Errorf("长度必须在%d到%d个字符之间", r.min, r.max)
	}

	return nil
}

// LengthRange 长度范围规则构造函数
func LengthRange(min, max int) ValidationRule {
	return &LengthRangeRule{min: min, max: max}
}

// EmailRule 邮箱规则
type EmailRule struct {
	regex *regexp2.Regexp
}

func (r *EmailRule) Validate(value any) error {
	str, ok := value.(string)
	if !ok || str == "" {
		return nil
	}

	matched, _ := r.regex.MatchString(str)
	if !matched {
		return fmt.Errorf("格式不正确")
	}

	return nil
}

// Email 邮箱规则构造函数
func Email() ValidationRule {
	return &EmailRule{
		regex: regexp2.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`, 0),
	}
}

// MobileRule 手机号规则
type MobileRule struct {
	regex *regexp2.Regexp
}

func (r *MobileRule) Validate(value any) error {
	str, ok := value.(string)
	if !ok || str == "" {
		return nil
	}

	matched, _ := r.regex.MatchString(str)
	if !matched {
		return fmt.Errorf("格式不正确")
	}

	return nil
}

// Mobile 手机号规则构造函数
func Mobile() ValidationRule {
	return &MobileRule{
		regex: regexp2.MustCompile(`^1[3-9]\d{9}$`, 0),
	}
}

// URLRule URL 规则
type URLRule struct {
	regex *regexp2.Regexp
}

func (r *URLRule) Validate(value any) error {
	str, ok := value.(string)
	if !ok || str == "" {
		return nil
	}

	matched, _ := r.regex.MatchString(str)
	if !matched {
		return fmt.Errorf("格式不正确")
	}

	return nil
}

// URL URL 规则构造函数
func URL() ValidationRule {
	return &URLRule{
		regex: regexp2.MustCompile(`^https?://[^\s]+$`, 0),
	}
}

// PatternRule 正则规则
type PatternRule struct {
	regex  *regexp2.Regexp
	errMsg string
}

func (r *PatternRule) Validate(value any) error {
	str, ok := value.(string)
	if !ok || str == "" {
		return nil
	}

	matched, _ := r.regex.MatchString(str)
	if !matched {
		if r.errMsg != "" {
			return fmt.Errorf("%s", r.errMsg)
		}
		return fmt.Errorf("格式不正确")
	}

	return nil
}

// Pattern 正则规则构造函数
func Pattern(pattern string, errMsg ...string) ValidationRule {
	msg := ""
	if len(errMsg) > 0 {
		msg = errMsg[0]
	}
	return &PatternRule{
		regex:  regexp2.MustCompile(pattern, 0),
		errMsg: msg,
	}
}

// InRule 枚举规则
type InRule struct {
	options []string
}

func (r *InRule) Validate(value any) error {
	str, ok := value.(string)
	if !ok || str == "" {
		return nil
	}

	for _, option := range r.options {
		if str == option {
			return nil
		}
	}

	return fmt.Errorf("的值不在允许的范围内")
}

// In 枚举规则构造函数
func In(options ...string) ValidationRule {
	return &InRule{options: options}
}

// RangeRule 数值范围规则
type RangeRule struct {
	min int
	max int
}

func (r *RangeRule) Validate(value any) error {
	var num int

	switch v := value.(type) {
	case int:
		num = v
	case int8:
		num = int(v)
	case int16:
		num = int(v)
	case int32:
		num = int(v)
	case int64:
		num = int(v)
	default:
		return nil
	}

	if num < r.min || num > r.max {
		return fmt.Errorf("必须在%d到%d之间", r.min, r.max)
	}

	return nil
}

// Range 数值范围规则构造函数
func Range(min, max int) ValidationRule {
	return &RangeRule{min: min, max: max}
}

// EqualsRule 相等规则
type EqualsRule struct {
	compareValue any
}

func (r *EqualsRule) Validate(value any) error {
	if value != r.compareValue {
		return fmt.Errorf("不一致")
	}
	return nil
}

// Equals 相等规则构造函数
func Equals(compareValue any) ValidationRule {
	return &EqualsRule{compareValue: compareValue}
}

// CustomRule 自定义规则
type CustomRule struct {
	validateFunc func(value any) error
}

func (r *CustomRule) Validate(value any) error {
	return r.validateFunc(value)
}

// Custom 自定义规则构造函数
func Custom(validateFunc func(value any) error) ValidationRule {
	return &CustomRule{validateFunc: validateFunc}
}

// CustomCondition 自定义条件规则构造函数
func CustomCondition(condition bool, errMsg string) ValidationRule {
	return &CustomRule{
		validateFunc: func(value any) error {
			if !condition {
				return fmt.Errorf("%s", errMsg)
			}
			return nil
		},
	}
}

// ============ 组合规则（组合模式） ============

// CompositeRule 组合规则
type CompositeRule struct {
	rules []ValidationRule
}

func (r *CompositeRule) Validate(value any) error {
	for _, rule := range r.rules {
		if err := rule.Validate(value); err != nil {
			return err
		}
	}
	return nil
}

// And 组合多个规则（所有规则都必须通过）
func And(rules ...ValidationRule) ValidationRule {
	return &CompositeRule{rules: rules}
}

// ============ 密码校验辅助函数 ============

// IsPassword 检查是否为有效密码（包含字母和数字）
func IsPassword(password string) bool {
	hasLetter := false
	hasDigit := false

	for _, char := range password {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
		} else if char >= '0' && char <= '9' {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}

	return hasLetter && hasDigit
}

// IsStrongPassword 检查是否为强密码（包含大小写字母、数字和特殊字符）
func IsStrongPassword(password string) bool {
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		if char >= 'a' && char <= 'z' {
			hasLower = true
		} else if char >= 'A' && char <= 'Z' {
			hasUpper = true
		} else if char >= '0' && char <= '9' {
			hasDigit = true
		} else if strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char) {
			hasSpecial = true
		}
	}

	return hasLower && hasUpper && hasDigit && hasSpecial
}

// ============ 便捷的校验规则组合 ============

// Username 用户名规则（字母、数字、下划线，4-20位）
func Username() ValidationRule {
	return And(
		Required(),
		LengthRange(4, 20),
		Pattern(`^[a-zA-Z0-9_]+$`, "只能包含字母、数字和下划线"),
	)
}

// Password 密码规则（至少包含字母和数字，6-20位）
func Password() ValidationRule {
	return And(
		Required(),
		LengthRange(6, 20),
		Custom(func(value any) error {
			str, ok := value.(string)
			if !ok {
				return nil
			}
			if !IsPassword(str) {
				return fmt.Errorf("必须包含字母和数字")
			}
			return nil
		}),
	)
}

// StrongPassword 强密码规则（大小写字母、数字、特殊字符，8-20位）
func StrongPassword() ValidationRule {
	return And(
		Required(),
		LengthRange(8, 20),
		Custom(func(value any) error {
			str, ok := value.(string)
			if !ok {
				return nil
			}
			if !IsStrongPassword(str) {
				return fmt.Errorf("必须包含大小写字母、数字和特殊字符")
			}
			return nil
		}),
	)
}

// ChineseName 中文姓名规则（2-4个汉字）
func ChineseName() ValidationRule {
	return And(
		Required(),
		Pattern(`^[\p{Han}]{2,4}$`, "必须为2-4个汉字"),
	)
}

// IDCard 身份证号规则
func IDCard() ValidationRule {
	return And(
		Required(),
		Pattern(`^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`, "格式不正确"),
	)
}

// xxx
