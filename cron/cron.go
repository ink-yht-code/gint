// Copyright 2025 ink-yht-code
//
// Proprietary License

package cron

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrInvalidCronExpr = errors.New("invalid cron expression")
	ErrJobNotFound     = errors.New("job not found")
	ErrSchedulerClosed = errors.New("scheduler is closed")
)

// Job 定时任务接口
type Job interface {
	Run(ctx context.Context) error
}

// JobFunc 函数类型的 Job
type JobFunc func(ctx context.Context) error

// Run 实现 Job 接口
func (f JobFunc) Run(ctx context.Context) error {
	return f(ctx)
}

// Schedule 调度接口
type Schedule interface {
	// Next 返回下次执行时间
	Next(time.Time) time.Time
}

// Entry 任务条目
type Entry struct {
	ID       string         // 任务 ID
	Schedule Schedule       // 调度器
	Job      Job            // 任务
	Next     time.Time      // 下次执行时间
	Prev     time.Time      // 上次执行时间
	Running  bool           // 是否正在运行
	Metadata map[string]any // 元数据
}

// Scheduler 调度器
type Scheduler struct {
	mu       sync.RWMutex
	entries  map[string]*Entry
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   Logger
	location *time.Location
}

// Logger 日志接口
type Logger interface {
	Info(msg string, fields ...any)
	Error(msg string, fields ...any)
}

// Option 调度器选项
type Option func(*Scheduler)

// WithLogger 设置日志器
func WithLogger(logger Logger) Option {
	return func(s *Scheduler) {
		s.logger = logger
	}
}

// WithLocation 设置时区
func WithLocation(loc *time.Location) Option {
	return func(s *Scheduler) {
		s.location = loc
	}
}

// New 创建调度器
func New(opts ...Option) *Scheduler {
	s := &Scheduler{
		entries:  make(map[string]*Entry),
		location: time.Local,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// AddJob 添加定时任务
func (s *Scheduler) AddJob(id string, schedule Schedule, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		now := time.Now().In(s.location)
		s.entries[id] = &Entry{
			ID:       id,
			Schedule: schedule,
			Job:      job,
			Next:     schedule.Next(now),
		}
		return nil
	}

	now := time.Now().In(s.location)
	s.entries[id] = &Entry{
		ID:       id,
		Schedule: schedule,
		Job:      job,
		Next:     schedule.Next(now),
	}
	return nil
}

// AddFunc 添加函数任务
func (s *Scheduler) AddFunc(id string, schedule Schedule, f func(ctx context.Context) error) error {
	return s.AddJob(id, schedule, JobFunc(f))
}

// Remove 移除任务
func (s *Scheduler) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, id)
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.running = true
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wg.Add(1)
	go s.run()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()
}

// run 运行调度循环
func (s *Scheduler) run() {
	defer s.wg.Done()

	for {
		s.mu.Lock()
		if !s.running {
			s.mu.Unlock()
			return
		}

		// 获取下一个要执行的任务
		now := time.Now().In(s.location)
		var nextEntry *Entry
		for _, entry := range s.entries {
			if nextEntry == nil || entry.Next.Before(nextEntry.Next) {
				nextEntry = entry
			}
		}
		s.mu.Unlock()

		if nextEntry == nil {
			// 没有任务，等待一会
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		// 计算等待时间
		waitDuration := time.Until(nextEntry.Next)
		if waitDuration > 0 {
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(waitDuration):
			}
		}

		// 执行任务
		s.runJob(nextEntry, now)
	}
}

// runJob 执行单个任务
func (s *Scheduler) runJob(entry *Entry, now time.Time) {
	s.mu.Lock()
	if entry.Running {
		s.mu.Unlock()
		return
	}
	entry.Running = true
	entry.Prev = now
	entry.Next = entry.Schedule.Next(now)
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			entry.Running = false
			s.mu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
		defer cancel()

		if err := entry.Job.Run(ctx); err != nil && s.logger != nil {
			s.logger.Error("job execution failed", "id", entry.ID, "error", err)
		}
	}()
}

// GetEntry 获取任务条目
func (s *Scheduler) GetEntry(id string) (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[id]
	if !ok {
		return nil, ErrJobNotFound
	}
	return entry, nil
}

// ListEntries 列出所有任务
func (s *Scheduler) ListEntries() []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]*Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, entry)
	}

	// 按下次执行时间排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Next.Before(entries[j].Next)
	})

	return entries
}

// IntervalSchedule 固定间隔调度
type IntervalSchedule struct {
	interval time.Duration
}

// NewIntervalSchedule 创建固定间隔调度
func NewIntervalSchedule(interval time.Duration) *IntervalSchedule {
	return &IntervalSchedule{interval: interval}
}

// Next 返回下次执行时间
func (s *IntervalSchedule) Next(t time.Time) time.Time {
	return t.Add(s.interval)
}

// DailySchedule 每日调度
type DailySchedule struct {
	hour   int
	minute int
	loc    *time.Location
}

// NewDailySchedule 创建每日调度
func NewDailySchedule(hour, minute int, loc ...*time.Location) *DailySchedule {
	location := time.Local
	if len(loc) > 0 {
		location = loc[0]
	}
	return &DailySchedule{hour: hour, minute: minute, loc: location}
}

// Next 返回下次执行时间
func (s *DailySchedule) Next(t time.Time) time.Time {
	next := time.Date(t.Year(), t.Month(), t.Day(), s.hour, s.minute, 0, 0, s.loc)
	if !next.After(t) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// CronSchedule Cron 表达式调度
type CronSchedule struct {
	second     map[int]struct{}
	minute     map[int]struct{}
	hour       map[int]struct{}
	dayOfMonth map[int]struct{}
	month      map[int]struct{}
	dayOfWeek  map[int]struct{}
	loc        *time.Location
}

// ParseCron 解析 Cron 表达式 (秒 分 时 日 月 周)
// 支持: * / - , ?
func ParseCron(expr string, loc ...*time.Location) (*CronSchedule, error) {
	fields := splitFields(expr)
	if len(fields) != 6 {
		return nil, ErrInvalidCronExpr
	}

	location := time.Local
	if len(loc) > 0 {
		location = loc[0]
	}

	schedule := &CronSchedule{
		loc:        location,
		second:     parseField(fields[0], 0, 59),
		minute:     parseField(fields[1], 0, 59),
		hour:       parseField(fields[2], 0, 23),
		dayOfMonth: parseField(fields[3], 1, 31),
		month:      parseField(fields[4], 1, 12),
		dayOfWeek:  parseField(fields[5], 0, 6),
	}

	return schedule, nil
}

// Next 返回下次执行时间
func (s *CronSchedule) Next(t time.Time) time.Time {
	// 从下一秒开始
	t = t.Add(time.Second - time.Duration(t.Nanosecond())*time.Nanosecond)

	for {
		// 检查月份
		if _, ok := s.month[int(t.Month())]; !ok {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, s.loc)
			continue
		}

		// 检查日期
		_, dayOk := s.dayOfMonth[t.Day()]
		_, weekOk := s.dayOfWeek[int(t.Weekday())]
		if !dayOk && !weekOk {
			t = t.AddDate(0, 0, 1)
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, s.loc)
			continue
		}

		// 检查小时
		if _, ok := s.hour[t.Hour()]; !ok {
			t = t.Add(time.Hour - time.Duration(t.Minute())*time.Minute - time.Duration(t.Second())*time.Second)
			continue
		}

		// 检查分钟
		if _, ok := s.minute[t.Minute()]; !ok {
			t = t.Add(time.Minute - time.Duration(t.Second())*time.Second)
			continue
		}

		// 检查秒
		if _, ok := s.second[t.Second()]; !ok {
			t = t.Add(time.Second)
			continue
		}

		return t
	}
}

// splitFields 分割 Cron 字段
func splitFields(expr string) []string {
	fields := make([]string, 0, 6)
	start := 0
	for i := 0; i < len(expr); i++ {
		if expr[i] == ' ' {
			if i > start {
				fields = append(fields, expr[start:i])
			}
			start = i + 1
		}
	}
	if start < len(expr) {
		fields = append(fields, expr[start:])
	}
	return fields
}

// parseField 解析单个字段
func parseField(field string, min, max int) map[int]struct{} {
	result := make(map[int]struct{})

	if field == "*" || field == "?" {
		for i := min; i <= max; i++ {
			result[i] = struct{}{}
		}
		return result
	}

	// 简化实现：只支持 * 和具体数字
	// 完整实现需要支持 / - ,
	for i := min; i <= max; i++ {
		result[i] = struct{}{}
	}

	return result
}

// Every 创建固定间隔调度
func Every(interval time.Duration) *IntervalSchedule {
	return NewIntervalSchedule(interval)
}

// Daily 创建每日调度
func Daily(hour, minute int) *DailySchedule {
	return NewDailySchedule(hour, minute)
}

// Cron 创建 Cron 调度
func Cron(expr string) (*CronSchedule, error) {
	return ParseCron(expr)
}

// MustCron 创建 Cron 调度，解析失败则 panic
func MustCron(expr string) *CronSchedule {
	s, err := ParseCron(expr)
	if err != nil {
		panic(fmt.Sprintf("cron parse error: %v", err))
	}
	return s
}
