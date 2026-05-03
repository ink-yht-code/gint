package sqlxutil

import "strings"

// ParamsMap 是 repository 层用于组织 named SQL 参数的统一类型。
// 它本质上仍然是 map[string]any，不引入额外反射或复杂构造成本。
type ParamsMap map[string]any

// Set 写入一个参数并返回当前 ParamsMap，便于链式或渐进式组装。
func (m ParamsMap) Set(key string, value any) ParamsMap {
	m[key] = value
	return m
}

// SetIf 在 condition 为 true 时写入参数。
func (m ParamsMap) SetIf(key string, value any, condition bool) ParamsMap {
	if condition {
		m[key] = value
	}
	return m
}

// SetIfNotEmpty 在字符串非空时写入参数，并自动做 TrimSpace。
func (m ParamsMap) SetIfNotEmpty(key, value string) ParamsMap {
	value = strings.TrimSpace(value)
	if value != "" {
		m[key] = value
	}
	return m
}
