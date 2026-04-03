// Copyright 2025 ink-yht-code
//
// Proprietary License

package mr

import (
	"context"
	"sync"
	"sync/atomic"
)

// MapFunc 映射函数
type MapFunc[T, U any] func(ctx context.Context, item T) (U, error)

// ReduceFunc 归约函数
type ReduceFunc[U, R any] func(ctx context.Context, acc R, item U) (R, error)

// FilterFunc 过滤函数
type FilterFunc[T any] func(ctx context.Context, item T) (bool, error)

// MapReduce 执行 MapReduce 操作
// source -> map -> reduce -> result
func MapReduce[T, U, R any](
	ctx context.Context,
	source []T,
	mapFunc MapFunc[T, U],
	reduceFunc ReduceFunc[U, R],
	opts ...Option,
) (R, error) {
	options := applyOptions(opts...)

	// 创建带缓冲的 channel
	mapOutput := make(chan U, options.bufferSize)
	var mapErrors []error
	var mapErrMu sync.Mutex

	// 并发执行 Map
	var wg sync.WaitGroup
	var activeCount int64

	// 使用信号量控制并发数
	sem := make(chan struct{}, options.concurrency)

	for _, item := range source {
		select {
		case <-ctx.Done():
			var zero R
			return zero, ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)
		atomic.AddInt64(&activeCount, 1)

		go func(item T) {
			defer wg.Done()
			defer atomic.AddInt64(&activeCount, -1)
			defer func() { <-sem }()

			result, err := mapFunc(ctx, item)
			if err != nil {
				mapErrMu.Lock()
				mapErrors = append(mapErrors, err)
				mapErrMu.Unlock()
				return
			}

			select {
			case mapOutput <- result:
			case <-ctx.Done():
			}
		}(item)
	}

	// 等待 Map 完成并关闭 channel
	go func() {
		wg.Wait()
		close(mapOutput)
	}()

	// 执行 Reduce
	var acc R
	var reduceErr error

	for item := range mapOutput {
		acc, reduceErr = reduceFunc(ctx, acc, item)
		if reduceErr != nil {
			break
		}
	}

	// 合并错误
	if len(mapErrors) > 0 {
		return acc, mapErrors[0]
	}

	return acc, reduceErr
}

// Map 并发映射
func Map[T, U any](ctx context.Context, source []T, mapFunc MapFunc[T, U], opts ...Option) ([]U, error) {
	options := applyOptions(opts...)

	results := make([]U, len(source))
	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	sem := make(chan struct{}, options.concurrency)

	for i, item := range source {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(i int, item T) {
			defer wg.Done()
			defer func() { <-sem }()

			result, err := mapFunc(ctx, item)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}

			results[i] = result
		}(i, item)
	}

	wg.Wait()
	return results, firstErr
}

// Filter 并发过滤
func Filter[T any](ctx context.Context, source []T, filterFunc FilterFunc[T], opts ...Option) ([]T, error) {
	options := applyOptions(opts...)

	results := make([]T, 0, len(source))
	var resultsMu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	sem := make(chan struct{}, options.concurrency)

	for _, item := range source {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(item T) {
			defer wg.Done()
			defer func() { <-sem }()

			ok, err := filterFunc(ctx, item)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}

			if ok {
				resultsMu.Lock()
				results = append(results, item)
				resultsMu.Unlock()
			}
		}(item)
	}

	wg.Wait()
	return results, firstErr
}

// MapReduceWithContext 带取消的 MapReduce
func MapReduceWithContext[T, U, R any](
	ctx context.Context,
	source []T,
	mapFunc MapFunc[T, U],
	reduceFunc ReduceFunc[U, R],
	opts ...Option,
) (R, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(ctx)
	result, err := MapReduce(ctx, source, mapFunc, reduceFunc, opts...)
	return result, cancel, err
}

// ForEach 并发遍历
func ForEach[T any](ctx context.Context, source []T, fn func(ctx context.Context, item T) error, opts ...Option) error {
	options := applyOptions(opts...)

	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	sem := make(chan struct{}, options.concurrency)

	for _, item := range source {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(item T) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := fn(ctx, item); err != nil {
				errOnce.Do(func() { firstErr = err })
			}
		}(item)
	}

	wg.Wait()
	return firstErr
}

// FlatMap 映射并展平
func FlatMap[T, U any](ctx context.Context, source []T, mapFunc func(ctx context.Context, item T) ([]U, error), opts ...Option) ([]U, error) {
	options := applyOptions(opts...)

	results := make([][]U, len(source))
	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	sem := make(chan struct{}, options.concurrency)

	for i, item := range source {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(i int, item T) {
			defer wg.Done()
			defer func() { <-sem }()

			result, err := mapFunc(ctx, item)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}

			results[i] = result
		}(i, item)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	// 展平结果
	var total int
	for _, r := range results {
		total += len(r)
	}

	flat := make([]U, 0, total)
	for _, r := range results {
		flat = append(flat, r...)
	}

	return flat, nil
}

// Reduce 顺序归约
func Reduce[T, R any](ctx context.Context, source []T, reduceFunc func(ctx context.Context, acc R, item T) (R, error), initial R) (R, error) {
	acc := initial
	var err error

	for _, item := range source {
		select {
		case <-ctx.Done():
			return acc, ctx.Err()
		default:
			acc, err = reduceFunc(ctx, acc, item)
			if err != nil {
				return acc, err
			}
		}
	}

	return acc, nil
}

// Partition 分区处理
func Partition[T any](ctx context.Context, source []T, size int, fn func(ctx context.Context, partition []T) error, opts ...Option) error {
	options := applyOptions(opts...)

	if size <= 0 {
		size = 100
	}

	partitions := make([][]T, 0, (len(source)+size-1)/size)

	for i := 0; i < len(source); i += size {
		end := i + size
		if end > len(source) {
			end = len(source)
		}
		partitions = append(partitions, source[i:end])
	}

	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	sem := make(chan struct{}, options.concurrency)

	for _, partition := range partitions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(p []T) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := fn(ctx, p); err != nil {
				errOnce.Do(func() { firstErr = err })
			}
		}(partition)
	}

	wg.Wait()
	return firstErr
}

// Options 配置选项
type Options struct {
	concurrency int
	bufferSize  int
}

// Option 配置函数
type Option func(*Options)

// WithConcurrency 设置并发数
func WithConcurrency(n int) Option {
	return func(o *Options) {
		if n > 0 {
			o.concurrency = n
		}
	}
}

// WithBufferSize 设置缓冲区大小
func WithBufferSize(n int) Option {
	return func(o *Options) {
		if n > 0 {
			o.bufferSize = n
		}
	}
}

func applyOptions(opts ...Option) *Options {
	o := &Options{
		concurrency: 16,
		bufferSize:  1000,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
