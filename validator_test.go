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

package gint

import (
	"fmt"
	"testing"
)

func TestRequired(t *testing.T) {
	rule := Required()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"nil value", nil, true},
		{"empty string", "", true},
		{"whitespace string", "   ", true},
		{"valid string", "test", false},
		{"zero int", 0, false},
		{"positive int", 123, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Required.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	rule := Email()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"empty string", "", false}, // empty is allowed
		{"valid email", "test@example.com", false},
		{"valid email with subdomain", "user@mail.example.com", false},
		{"invalid email - no @", "testexample.com", true},
		{"invalid email - no domain", "test@", true},
		{"invalid email - no TLD", "test@example", true},
		{"non-string value", 123, false}, // non-string is allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Email.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMobile(t *testing.T) {
	rule := Mobile()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"empty string", "", false}, // empty is allowed
		{"valid mobile", "13812345678", false},
		{"valid mobile - 15x", "15912345678", false},
		{"valid mobile - 18x", "18812345678", false},
		{"invalid mobile - too short", "1381234567", true},
		{"invalid mobile - too long", "138123456789", true},
		{"invalid mobile - wrong prefix", "12812345678", true},
		{"invalid mobile - contains letters", "1381234567a", true},
		{"non-string value", 123, false}, // non-string is allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mobile.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLengthRange(t *testing.T) {
	rule := LengthRange(3, 10)

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"too short", "ab", true},
		{"minimum length", "abc", false},
		{"middle length", "abcdef", false},
		{"maximum length", "abcdefghij", false},
		{"too long", "abcdefghijk", true},
		{"chinese characters", "你好世界", false}, // 4 characters
		{"non-string value", 123, false},      // non-string is allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("LengthRange.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRange(t *testing.T) {
	rule := Range(1, 100)

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"below range", 0, true},
		{"minimum value", 1, false},
		{"middle value", 50, false},
		{"maximum value", 100, false},
		{"above range", 101, true},
		{"int8", int8(50), false},
		{"int16", int16(50), false},
		{"int32", int32(50), false},
		{"int64", int64(50), false},
		{"non-numeric value", "50", false}, // non-numeric is allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Range.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIn(t *testing.T) {
	rule := In("apple", "banana", "orange")

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"empty string", "", false}, // empty is allowed
		{"valid option 1", "apple", false},
		{"valid option 2", "banana", false},
		{"valid option 3", "orange", false},
		{"invalid option", "grape", true},
		{"non-string value", 123, false}, // non-string is allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("In.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPattern(t *testing.T) {
	rule := Pattern(`^\d{3}-\d{4}$`, "格式应为 XXX-XXXX")

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"empty string", "", false}, // empty is allowed
		{"valid pattern", "123-4567", false},
		{"invalid pattern - no dash", "1234567", true},
		{"invalid pattern - wrong format", "12-34567", true},
		{"non-string value", 123, false}, // non-string is allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Pattern.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustom(t *testing.T) {
	rule := Custom(func(value any) error {
		str, ok := value.(string)
		if !ok {
			return nil
		}
		if str == "forbidden" {
			return fmt.Errorf("this value is forbidden")
		}
		return nil
	})

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"allowed string", "allowed", false},
		{"forbidden string", "forbidden", true},
		{"non-string value", 123, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Custom.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnd(t *testing.T) {
	rule := And(
		Required(),
		LengthRange(3, 10),
		Pattern(`^[a-zA-Z]+$`, "只能包含字母"),
	)

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"valid value", "hello", false},
		{"empty value", "", true},              // fails Required
		{"too short", "ab", true},              // fails LengthRange
		{"too long", "abcdefghijk", true},      // fails LengthRange
		{"contains numbers", "hello123", true}, // fails Pattern
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("And.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatorBuilder(t *testing.T) {
	vb := NewValidatorBuilder()

	vb.Field("用户名", "").AddRule(Required()).AddRule(Username())
	vb.Field("邮箱", "invalid-email").AddRule(Required()).AddRule(Email())
	vb.Field("年龄", 150).AddRule(Range(1, 120))

	vb.Validate()

	if vb.IsValid() {
		t.Error("Expected validation to fail, but it passed")
	}

	errors := vb.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected validation errors, but got none")
	}

	firstError := vb.GetFirstError()
	if firstError == "" {
		t.Error("Expected first error, but got empty string")
	}

	errorString := vb.GetErrorString()
	if errorString == "" {
		t.Error("Expected error string, but got empty string")
	}
}

func TestIsPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"valid password", "abc123", true},
		{"only letters", "abcdef", false},
		{"only numbers", "123456", false},
		{"empty string", "", false},
		{"complex password", "Abc123!@#", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPassword(tt.password); got != tt.want {
				t.Errorf("IsPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStrongPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"strong password", "Abc123!@#", true},
		{"missing lowercase", "ABC123!@#", false},
		{"missing uppercase", "abc123!@#", false},
		{"missing digit", "Abc!@#def", false},
		{"missing special", "Abc123def", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStrongPassword(tt.password); got != tt.want {
				t.Errorf("IsStrongPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPresetRules(t *testing.T) {
	tests := []struct {
		name  string
		rule  ValidationRule
		value string
		valid bool
	}{
		{"Username - valid", Username(), "user123", true},
		{"Username - too short", Username(), "ab", false},
		{"Username - invalid chars", Username(), "user@123", false},
		{"Password - valid", Password(), "abc123", true},
		{"Password - too short", Password(), "ab1", false},
		{"Password - no digits", Password(), "abcdef", false},
		{"StrongPassword - valid", StrongPassword(), "Abc123!@#", true},
		{"StrongPassword - weak", StrongPassword(), "abc123", false},
		{"ChineseName - valid", ChineseName(), "张三", true},
		{"ChineseName - too short", ChineseName(), "张", false},
		{"ChineseName - contains english", ChineseName(), "张san", false},
		{"IDCard - valid", IDCard(), "110101199001011234", true},
		{"IDCard - invalid", IDCard(), "123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate(tt.value)
			isValid := err == nil
			if isValid != tt.valid {
				t.Errorf("Rule validation = %v, want %v, error: %v", isValid, tt.valid, err)
			}
		})
	}
}
