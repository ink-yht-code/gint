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

package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Config 数据库配置
type Config struct {
	// Driver 驱动名称（mysql, postgres, sqlite等）
	Driver string
	// DSN 数据源名称
	DSN string
	// MaxOpenConns 最大打开连接数
	MaxOpenConns int
	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int
	// ConnMaxLifetime 连接最大存活时间
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime 空闲连接最大存活时间
	ConnMaxIdleTime time.Duration
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		Driver:          "mysql",
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// DB 数据库封装
type DB struct {
	*sql.DB
	config Config
}

// Open 打开数据库连接
func Open(config Config) (*DB, error) {
	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, err
	}

	// 设置连接池参数
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	return &DB{DB: db, config: config}, nil
}

// OpenMySQL 打开 MySQL 连接
func OpenMySQL(dsn string, config ...Config) (*DB, error) {
	cfg := DefaultConfig()
	cfg.Driver = "mysql"
	cfg.DSN = dsn
	if len(config) > 0 {
		cfg = config[0]
		cfg.Driver = "mysql"
		cfg.DSN = dsn
	}
	return Open(cfg)
}

// Ping 检查连接
func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

// Stats 获取连接池统计
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// Close 关闭连接
func (db *DB) Close() error {
	return db.DB.Close()
}

// --- 查询构建器 ---

// QueryBuilder 查询构建器
type QueryBuilder struct {
	table   string
	fields  []string
	where   []string
	args    []any
	orderBy string
	limit   int
	offset  int
	groupBy string
	having  string
}

// Table 创建查询构建器
func (db *DB) Table(table string) *QueryBuilder {
	return &QueryBuilder{table: table}
}

// Select 设置查询字段
func (q *QueryBuilder) Select(fields ...string) *QueryBuilder {
	q.fields = fields
	return q
}

// Where 添加 WHERE 条件
func (q *QueryBuilder) Where(condition string, args ...any) *QueryBuilder {
	q.where = append(q.where, condition)
	q.args = append(q.args, args...)
	return q
}

// WhereIn 添加 IN 条件
func (q *QueryBuilder) WhereIn(field string, values []any) *QueryBuilder {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}
	q.where = append(q.where, fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ",")))
	q.args = append(q.args, values...)
	return q
}

// WhereLike 添加 LIKE 条件
func (q *QueryBuilder) WhereLike(field, pattern string) *QueryBuilder {
	q.where = append(q.where, fmt.Sprintf("%s LIKE ?", field))
	q.args = append(q.args, pattern)
	return q
}

// OrderBy 设置排序
func (q *QueryBuilder) OrderBy(order string) *QueryBuilder {
	q.orderBy = order
	return q
}

// Limit 设置限制
func (q *QueryBuilder) Limit(limit int) *QueryBuilder {
	q.limit = limit
	return q
}

// Offset 设置偏移
func (q *QueryBuilder) Offset(offset int) *QueryBuilder {
	q.offset = offset
	return q
}

// GroupBy 设置分组
func (q *QueryBuilder) GroupBy(group string) *QueryBuilder {
	q.groupBy = group
	return q
}

// Having 设置 HAVING 条件
func (q *QueryBuilder) Having(condition string, args ...any) *QueryBuilder {
	q.having = condition
	q.args = append(q.args, args...)
	return q
}

// Build 构建 SQL
func (q *QueryBuilder) Build() (string, []any) {
	var sql strings.Builder

	// SELECT
	sql.WriteString("SELECT ")
	if len(q.fields) > 0 {
		sql.WriteString(strings.Join(q.fields, ", "))
	} else {
		sql.WriteString("*")
	}

	// FROM
	sql.WriteString(" FROM ")
	sql.WriteString(q.table)

	// WHERE
	if len(q.where) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(q.where, " AND "))
	}

	// GROUP BY
	if q.groupBy != "" {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(q.groupBy)
	}

	// HAVING
	if q.having != "" {
		sql.WriteString(" HAVING ")
		sql.WriteString(q.having)
	}

	// ORDER BY
	if q.orderBy != "" {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(q.orderBy)
	}

	// LIMIT
	if q.limit > 0 {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", q.limit))
	}

	// OFFSET
	if q.offset > 0 {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", q.offset))
	}

	return sql.String(), q.args
}

// Get 执行查询获取多行
func (q *QueryBuilder) Get(ctx context.Context, db *DB) (*sql.Rows, error) {
	sql, args := q.Build()
	return db.QueryContext(ctx, sql, args...)
}

// First 获取第一行
func (q *QueryBuilder) First(ctx context.Context, db *DB) *sql.Row {
	q.limit = 1
	sql, args := q.Build()
	return db.QueryRowContext(ctx, sql, args...)
}

// Count 计数
func (q *QueryBuilder) Count(ctx context.Context, db *DB) (int64, error) {
	q.fields = []string{"COUNT(*)"}
	sql, args := q.Build()
	var count int64
	err := db.QueryRowContext(ctx, sql, args...).Scan(&count)
	return count, err
}

// --- 插入构建器 ---

// InsertBuilder 插入构建器
type InsertBuilder struct {
	table  string
	fields []string
	values [][]any
}

// Insert 创建插入构建器
func (db *DB) Insert(table string) *InsertBuilder {
	return &InsertBuilder{table: table}
}

// Fields 设置字段
func (b *InsertBuilder) Fields(fields ...string) *InsertBuilder {
	b.fields = fields
	return b
}

// Values 添加值
func (b *InsertBuilder) Values(values ...any) *InsertBuilder {
	b.values = append(b.values, values)
	return b
}

// Build 构建 SQL
func (b *InsertBuilder) Build() (string, []any) {
	var sql strings.Builder
	var args []any

	sql.WriteString("INSERT INTO ")
	sql.WriteString(b.table)

	// 字段
	if len(b.fields) > 0 {
		sql.WriteString(" (")
		sql.WriteString(strings.Join(b.fields, ", "))
		sql.WriteString(")")
	}

	// 值
	sql.WriteString(" VALUES ")
	placeholders := make([]string, len(b.values))
	for i, vals := range b.values {
		ph := make([]string, len(vals))
		for j := range vals {
			ph[j] = "?"
		}
		placeholders[i] = "(" + strings.Join(ph, ",") + ")"
		args = append(args, vals...)
	}
	sql.WriteString(strings.Join(placeholders, ","))

	return sql.String(), args
}

// Exec 执行插入
func (b *InsertBuilder) Exec(ctx context.Context, db *DB) (sql.Result, error) {
	sql, args := b.Build()
	return db.ExecContext(ctx, sql, args...)
}

// --- 更新构建器 ---

// UpdateBuilder 更新构建器
type UpdateBuilder struct {
	table string
	sets  map[string]any
	where []string
	args  []any
}

// Update 创建更新构建器
func (db *DB) Update(table string) *UpdateBuilder {
	return &UpdateBuilder{table: table, sets: make(map[string]any)}
}

// Set 设置字段值
func (b *UpdateBuilder) Set(field string, value any) *UpdateBuilder {
	b.sets[field] = value
	return b
}

// Sets 批量设置字段值
func (b *UpdateBuilder) Sets(sets map[string]any) *UpdateBuilder {
	for k, v := range sets {
		b.sets[k] = v
	}
	return b
}

// Where 添加 WHERE 条件
func (b *UpdateBuilder) Where(condition string, args ...any) *UpdateBuilder {
	b.where = append(b.where, condition)
	b.args = append(b.args, args...)
	return b
}

// Build 构建 SQL
func (b *UpdateBuilder) Build() (string, []any) {
	var sql strings.Builder
	var args []any

	sql.WriteString("UPDATE ")
	sql.WriteString(b.table)
	sql.WriteString(" SET ")

	// SET
	setStrs := make([]string, 0, len(b.sets))
	for k, v := range b.sets {
		setStrs = append(setStrs, fmt.Sprintf("%s = ?", k))
		args = append(args, v)
	}
	sql.WriteString(strings.Join(setStrs, ", "))

	// WHERE
	if len(b.where) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(b.where, " AND "))
		args = append(args, b.args...)
	}

	return sql.String(), args
}

// Exec 执行更新
func (b *UpdateBuilder) Exec(ctx context.Context, db *DB) (sql.Result, error) {
	sql, args := b.Build()
	return db.ExecContext(ctx, sql, args...)
}

// --- 删除构建器 ---

// DeleteBuilder 删除构建器
type DeleteBuilder struct {
	table string
	where []string
	args  []any
}

// Delete 创建删除构建器
func (db *DB) Delete(table string) *DeleteBuilder {
	return &DeleteBuilder{table: table}
}

// Where 添加 WHERE 条件
func (b *DeleteBuilder) Where(condition string, args ...any) *DeleteBuilder {
	b.where = append(b.where, condition)
	b.args = append(b.args, args...)
	return b
}

// Build 构建 SQL
func (b *DeleteBuilder) Build() (string, []any) {
	var sql strings.Builder

	sql.WriteString("DELETE FROM ")
	sql.WriteString(b.table)

	if len(b.where) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(b.where, " AND "))
	}

	return sql.String(), b.args
}

// Exec 执行删除
func (b *DeleteBuilder) Exec(ctx context.Context, db *DB) (sql.Result, error) {
	sql, args := b.Build()
	return db.ExecContext(ctx, sql, args...)
}

// --- 扫描辅助 ---

// Scan 扫描行到结构体
func Scan[T any](rows *sql.Rows) ([]T, error) {
	var results []T

	for rows.Next() {
		var item T
		vals, err := scanValues(&item)
		if err != nil {
			return nil, err
		}
		if err := rows.Scan(vals...); err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	return results, rows.Err()
}

// ScanOne 扫描单行到结构体
func ScanOne[T any](row *sql.Row) (*T, error) {
	var item T
	vals, err := scanValues(&item)
	if err != nil {
		return nil, err
	}
	if err := row.Scan(vals...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// scanValues 获取结构体字段指针
func scanValues(dest any) ([]any, error) {
	v := reflect.ValueOf(dest).Elem()
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("dest must be a struct")
	}

	vals := make([]any, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// 跳过未导出字段
		if field.PkgPath != "" {
			continue
		}
		vals = append(vals, v.Field(i).Addr().Interface())
	}

	return vals, nil
}

// --- 事务 ---

// Tx 事务
type Tx struct {
	*sql.Tx
	db *DB
}

// Begin 开始事务
func (db *DB) Begin(ctx context.Context, opts ...*sql.TxOptions) (*Tx, error) {
	var opt *sql.TxOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	tx, err := db.BeginTx(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, db: db}, nil
}

// Table 在事务中创建查询构建器
func (tx *Tx) Table(table string) *QueryBuilder {
	return &QueryBuilder{table: table}
}

// Commit 提交事务
func (tx *Tx) Commit() error {
	return tx.Tx.Commit()
}

// Rollback 回滚事务
func (tx *Tx) Rollback() error {
	return tx.Tx.Rollback()
}

// Transaction 执行事务
func (db *DB) Transaction(ctx context.Context, fn func(tx *Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// --- 全局数据库 ---

var defaultDB *DB

// SetDefault 设置默认数据库
func SetDefault(db *DB) {
	defaultDB = db
}

// Default 获取默认数据库
func Default() *DB {
	return defaultDB
}
