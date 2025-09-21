package scheduler

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CronScheduler struct {
	jobs     map[string]*Job
	jobsMux  sync.RWMutex
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
}

type Job struct {
	ID          string
	Name        string
	Schedule    *Schedule
	Handler     func() error
	LastRun     time.Time
	NextRun     time.Time
	Running     bool
	Enabled     bool
	ErrorCount  int
	LastError   error
	RunCount    int
	mutex       sync.Mutex
}

type Schedule struct {
	Minutes  []int
	Hours    []int
	Days     []int
	Months   []int
	Weekdays []int
}

func NewCronScheduler() *CronScheduler {
	return &CronScheduler{
		jobs:     make(map[string]*Job),
		stopChan: make(chan struct{}),
	}
}

func (cs *CronScheduler) Start() error {
	cs.jobsMux.Lock()
	defer cs.jobsMux.Unlock()

	if cs.running {
		return fmt.Errorf("scheduler already running")
	}

	cs.running = true
	cs.stopChan = make(chan struct{})

	cs.wg.Add(1)
	go cs.runLoop()

	log.Println("🕐 Cron планировщик запущен")
	return nil
}

func (cs *CronScheduler) Stop() {
	cs.jobsMux.Lock()
	if !cs.running {
		cs.jobsMux.Unlock()
		return
	}
	cs.running = false
	cs.jobsMux.Unlock()

	close(cs.stopChan)
	cs.wg.Wait()

	log.Println("🛑 Cron планировщик остановлен")
}

func (cs *CronScheduler) AddJob(id, name, cronExpr string, handler func() error) error {
	schedule, err := ParseCronExpression(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	job := &Job{
		ID:       id,
		Name:     name,
		Schedule: schedule,
		Handler:  handler,
		Enabled:  true,
	}

	job.NextRun = cs.calculateNextRun(job.Schedule, time.Now())

	cs.jobsMux.Lock()
	cs.jobs[id] = job
	cs.jobsMux.Unlock()

	log.Printf("➕ Добавлено задание '%s' (%s) со расписанием: %s", name, id, cronExpr)
	log.Printf("🕒 Следующий запуск: %s", job.NextRun.Format("2006-01-02 15:04:05"))

	return nil
}

func (cs *CronScheduler) RemoveJob(id string) {
	cs.jobsMux.Lock()
	defer cs.jobsMux.Unlock()

	if job, exists := cs.jobs[id]; exists {
		delete(cs.jobs, id)
		log.Printf("➖ Удалено задание '%s' (%s)", job.Name, id)
	}
}

func (cs *CronScheduler) EnableJob(id string) error {
	cs.jobsMux.Lock()
	defer cs.jobsMux.Unlock()

	job, exists := cs.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Enabled = true
	job.NextRun = cs.calculateNextRun(job.Schedule, time.Now())
	log.Printf("✅ Задание '%s' включено", job.Name)
	return nil
}

func (cs *CronScheduler) DisableJob(id string) error {
	cs.jobsMux.Lock()
	defer cs.jobsMux.Unlock()

	job, exists := cs.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Enabled = false
	log.Printf("❌ Задание '%s' отключено", job.Name)
	return nil
}

func (cs *CronScheduler) GetJobs() map[string]*Job {
	cs.jobsMux.RLock()
	defer cs.jobsMux.RUnlock()

	result := make(map[string]*Job)
	for id, job := range cs.jobs {
		jobCopy := *job
		result[id] = &jobCopy
	}

	return result
}

func (cs *CronScheduler) runLoop() {
	defer cs.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("🔄 Запущен основной цикл планировщика")

	for {
		select {
		case <-cs.stopChan:
			return
		case now := <-ticker.C:
			cs.checkAndRunJobs(now)
		}
	}
}

func (cs *CronScheduler) checkAndRunJobs(now time.Time) {
	cs.jobsMux.RLock()
	var jobsToRun []*Job

	for _, job := range cs.jobs {
		if job.Enabled && !job.Running && now.After(job.NextRun) {
			jobsToRun = append(jobsToRun, job)
		}
	}
	cs.jobsMux.RUnlock()

	for _, job := range jobsToRun {
		go cs.runJob(job, now)
	}
}

func (cs *CronScheduler) runJob(job *Job, now time.Time) {
	job.mutex.Lock()
	if job.Running {
		job.mutex.Unlock()
		return
	}

	job.Running = true
	job.LastRun = now
	job.RunCount++
	job.mutex.Unlock()

	log.Printf("🚀 Запуск задания '%s' (%s)", job.Name, job.ID)
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- job.Handler()
	}()

	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		err = fmt.Errorf("job timeout after 30 minutes")
	}

	duration := time.Since(startTime)

	job.mutex.Lock()
	job.Running = false
	job.LastError = err
	if err != nil {
		job.ErrorCount++
		log.Printf("❌ Задание '%s' завершилось с ошибкой за %v: %v", job.Name, duration, err)
	} else {
		log.Printf("✅ Задание '%s' выполнено успешно за %v", job.Name, duration)
	}

	job.NextRun = cs.calculateNextRun(job.Schedule, now)
	log.Printf("🕒 Следующий запуск задания '%s': %s", job.Name, job.NextRun.Format("2006-01-02 15:04:05"))
	job.mutex.Unlock()
}

func (cs *CronScheduler) calculateNextRun(schedule *Schedule, from time.Time) time.Time {

	next := from.Add(time.Minute).Truncate(time.Minute)

	for i := 0; i < 366*24*60; i++ {
		if cs.matchesSchedule(schedule, next) {
			return next
		}
		next = next.Add(time.Minute)
	}

	return from.Add(365 * 24 * time.Hour)
}

func (cs *CronScheduler) matchesSchedule(schedule *Schedule, t time.Time) bool {
	minute := t.Minute()
	hour := t.Hour()
	day := t.Day()
	month := int(t.Month())
	weekday := int(t.Weekday())

	if len(schedule.Minutes) > 0 && !contains(schedule.Minutes, minute) {
		return false
	}
	if len(schedule.Hours) > 0 && !contains(schedule.Hours, hour) {
		return false
	}
	if len(schedule.Days) > 0 && !contains(schedule.Days, day) {
		return false
	}
	if len(schedule.Months) > 0 && !contains(schedule.Months, month) {
		return false
	}
	if len(schedule.Weekdays) > 0 && !contains(schedule.Weekdays, weekday) {
		return false
	}

	return true
}

func ParseCronExpression(expr string) (*Schedule, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 parts, got %d", len(parts))
	}

	schedule := &Schedule{}
	var err error

	schedule.Minutes, err = parseField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minutes: %w", err)
	}

	schedule.Hours, err = parseField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hours: %w", err)
	}

	schedule.Days, err = parseField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid days: %w", err)
	}

	schedule.Months, err = parseField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid months: %w", err)
	}

	schedule.Weekdays, err = parseField(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid weekdays: %w", err)
	}

	return schedule, nil
}

func parseField(field string, min, max int) ([]int, error) {
	if field == "*" {
		return nil, nil
	}

	var values []int

	parts := strings.Split(field, ",")
	for _, part := range parts {
		if strings.Contains(part, "/") {
			stepParts := strings.Split(part, "/")
			if len(stepParts) != 2 {
				return nil, fmt.Errorf("invalid step format: %s", part)
			}

			var start, end int
			if stepParts[0] == "*" {
				start, end = min, max
			} else if strings.Contains(stepParts[0], "-") {
				rangeParts := strings.Split(stepParts[0], "-")
				if len(rangeParts) != 2 {
					return nil, fmt.Errorf("invalid range format: %s", stepParts[0])
				}
				var err error
				start, err = strconv.Atoi(rangeParts[0])
				if err != nil {
					return nil, err
				}
				end, err = strconv.Atoi(rangeParts[1])
				if err != nil {
					return nil, err
				}
			} else {
				var err error
				start, err = strconv.Atoi(stepParts[0])
				if err != nil {
					return nil, err
				}
				end = max
			}

			step, err := strconv.Atoi(stepParts[1])
			if err != nil {
				return nil, err
			}

			for i := start; i <= end; i += step {
				if i >= min && i <= max {
					values = append(values, i)
				}
			}
		} else if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, err
			}

			for i := start; i <= end; i++ {
				if i >= min && i <= max {
					values = append(values, i)
				}
			}
		} else {
			value, err := strconv.Atoi(part)
			if err != nil {
				return nil, err
			}
			if value >= min && value <= max {
				values = append(values, value)
			}
		}
	}

	values = removeDuplicates(values)
	sort.Ints(values)

	return values, nil
}

func (cs *CronScheduler) GetJobStatus(id string) map[string]interface{} {
	cs.jobsMux.RLock()
	defer cs.jobsMux.RUnlock()

	job, exists := cs.jobs[id]
	if !exists {
		return map[string]interface{}{"error": "job not found"}
	}

	job.mutex.Lock()
	defer job.mutex.Unlock()

	status := map[string]interface{}{
		"id":         job.ID,
		"name":       job.Name,
		"enabled":    job.Enabled,
		"running":    job.Running,
		"run_count":  job.RunCount,
		"error_count": job.ErrorCount,
		"next_run":   job.NextRun.Format("2006-01-02 15:04:05"),
	}

	if !job.LastRun.IsZero() {
		status["last_run"] = job.LastRun.Format("2006-01-02 15:04:05")
	}

	if job.LastError != nil {
		status["last_error"] = job.LastError.Error()
	}

	return status
}

func contains(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func removeDuplicates(slice []int) []int {
	keys := make(map[int]bool)
	var result []int

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
