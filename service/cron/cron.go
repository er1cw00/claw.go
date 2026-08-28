package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/model"
	bus "github.com/er1cw00/claw.go/service/bus"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

// ScheduleKind defines the type of cron schedule.
type ScheduleKind string

const (
	ScheduleKindAt    ScheduleKind = "at"
	ScheduleKindEvery ScheduleKind = "every"
	ScheduleKindCron  ScheduleKind = "cron"
)

// CronSchedule is the schedule definition for a cron job.
type CronSchedule struct {
	Kind    ScheduleKind `json:"kind"`
	AtMs    int64        `json:"atMs,omitempty"`
	EveryMs int64        `json:"everyMs,omitempty"`
	Expr    string       `json:"expr,omitempty"`
	Tz      string       `json:"tz,omitempty"`
}

// CronPayload is what to do when the job runs.
type CronPayload struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Deliver bool   `json:"deliver"`
	Channel string `json:"channel,omitempty"`
	To      string `json:"to,omitempty"`
}

// CronJobState is the runtime state of a job.
type CronJobState struct {
	NextRunAtMs int64   `json:"nextRunAtMs,omitempty"`
	LastRunAtMs int64   `json:"lastRunAtMs,omitempty"`
	LastStatus  string  `json:"lastStatus,omitempty"`
	LastError   *string `json:"lastError,omitempty"`
}

// CronJob is a scheduled job.
type CronJob struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Enabled        bool         `json:"enabled"`
	Schedule       CronSchedule `json:"schedule"`
	Payload        CronPayload  `json:"payload"`
	State          CronJobState `json:"state"`
	CreatedAtMs    int64        `json:"createdAtMs"`
	UpdatedAtMs    int64        `json:"updatedAtMs"`
	DeleteAfterRun bool         `json:"deleteAfterRun"`
}

// CronStore is the persistent store for cron jobs.
type CronStore struct {
	Version int       `json:"version"`
	Jobs    []CronJob `json:"jobs"`
}

// Service manages and executes scheduled cron jobs.
type Service struct {
	mu        sync.RWMutex
	scheduler gocron.Scheduler
	store     *CronStore
	storePath string
	running   bool
	jobs      map[string]gocron.Job
}

var cronService *Service = &Service{}

// GetService returns the cron service instance.
func GetService() *Service {
	return cronService
}

// Start initializes the cron service, loads persisted jobs and starts the scheduler.
func (s *Service) Start() error {
	s.storePath = filepath.Join(base.GetSettings().Workspace, "cron.json")
	s.loadStore()

	var err error
	s.scheduler, err = gocron.NewScheduler()
	if err != nil {
		logger.Errorf("[Cron] start gocron scheduler fail, err: %v", err)
		return err
	}

	for i := range s.store.Jobs {
		job := &s.store.Jobs[i]
		if !job.Enabled {
			continue
		}
		if _, err := s.scheduleJob(job); err != nil {
			logger.Warnf("[Cron] schedule job %s (%s) failed: %v", job.Name, job.ID, err)
			job.State.LastStatus = "error"
			errMsg := err.Error()
			job.State.LastError = &errMsg
		}
	}

	s.scheduler.Start()
	s.running = true
	logger.Infof("[Cron] Service Start with %d jobs", len(s.store.Jobs))
	return nil
}

// Stop shuts down the cron scheduler.
func (s *Service) Stop() {
	if s.scheduler != nil {
		_ = s.scheduler.Shutdown()
		s.scheduler = nil
	}
	s.running = false
	logger.Info("[Cron] Service Stop")
}

// Name returns the service name.
func (s *Service) Name() string {
	return "Cron"
}

// ListJobs returns all jobs, optionally including disabled ones.
func (s *Service) ListJobs(includeDisabled bool) []CronJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]CronJob, 0, len(s.store.Jobs))
	for _, j := range s.store.Jobs {
		if !includeDisabled && !j.Enabled {
			continue
		}
		out = append(out, j)
	}
	sort.Slice(out, func(i, j int) bool {
		a := out[i].State.NextRunAtMs
		b := out[j].State.NextRunAtMs
		if a == 0 {
			return false
		}
		if b == 0 {
			return true
		}
		return a < b
	})
	return out
}

// AddJob adds a new cron job.
func (s *Service) AddJob(name string, schedule CronSchedule, message string, deliver bool, channel, to string, deleteAfterRun bool) (*CronJob, error) {
	now := nowMs()
	job := &CronJob{
		ID:      s.newID(),
		Name:    name,
		Enabled: true,
		Schedule: CronSchedule{
			Kind:    schedule.Kind,
			AtMs:    schedule.AtMs,
			EveryMs: schedule.EveryMs,
			Expr:    schedule.Expr,
			Tz:      schedule.Tz,
		},
		Payload: CronPayload{
			Kind:    "agent_turn",
			Message: message,
			Deliver: deliver,
			Channel: channel,
			To:      to,
		},
		State: CronJobState{
			NextRunAtMs: computeNextRun(schedule, now),
		},
		CreatedAtMs:    now,
		UpdatedAtMs:    now,
		DeleteAfterRun: deleteAfterRun,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		if _, err := s.scheduleJob(job); err != nil {
			return nil, err
		}
	}

	s.store.Jobs = append(s.store.Jobs, *job)
	s.saveStoreLocked()
	logger.Infof("[Cron] added job '%s' (%s)", name, job.ID)
	return job, nil
}

// RemoveJob removes a job by ID.
func (s *Service) RemoveJob(jobID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.store.Jobs)
	s.store.Jobs = filterJobs(s.store.Jobs, func(j CronJob) bool { return j.ID != jobID })
	removed := len(s.store.Jobs) < before

	if removed {
		if gJob, ok := s.jobs[jobID]; ok && s.scheduler != nil {
			_ = s.scheduler.RemoveJob(gJob.ID())
			delete(s.jobs, jobID)
		}
		s.saveStoreLocked()
		logger.Infof("[Cron] removed job %s", jobID)
	}
	return removed
}

// EnableJob enables or disables a job.
func (s *Service) EnableJob(jobID string, enabled bool) *CronJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.store.Jobs {
		job := &s.store.Jobs[i]
		if job.ID != jobID {
			continue
		}
		job.Enabled = enabled
		job.UpdatedAtMs = nowMs()
		if enabled {
			job.State.NextRunAtMs = computeNextRun(job.Schedule, nowMs())
			if s.running {
				if _, err := s.scheduleJob(job); err != nil {
					logger.Warnf("[Cron] re-schedule job %s failed: %v", jobID, err)
				}
			}
		} else {
			job.State.NextRunAtMs = 0
			if gJob, ok := s.jobs[jobID]; ok && s.scheduler != nil {
				_ = s.scheduler.RemoveJob(gJob.ID())
				delete(s.jobs, jobID)
			}
		}
		s.saveStoreLocked()
		return job
	}
	return nil
}

// RunJob manually runs a job by ID.
func (s *Service) RunJob(jobID string, force bool) bool {
	s.mu.RLock()
	found := -1
	for i, j := range s.store.Jobs {
		if j.ID == jobID {
			found = i
			break
		}
	}
	s.mu.RUnlock()
	if found < 0 {
		return false
	}

	job := &s.store.Jobs[found]
	if !job.Enabled && !force {
		return false
	}
	s.executeJob(job)
	s.mu.Lock()
	s.saveStoreLocked()
	s.mu.Unlock()
	return true
}

// Status returns the service status.
func (s *Service) Status() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nextWake := int64(0)
	for _, j := range s.store.Jobs {
		if !j.Enabled || j.State.NextRunAtMs == 0 {
			continue
		}
		if nextWake == 0 || j.State.NextRunAtMs < nextWake {
			nextWake = j.State.NextRunAtMs
		}
	}
	return map[string]any{
		"enabled":       s.running,
		"jobs":          len(s.store.Jobs),
		"nextWakeAtMs":  nextWake,
	}
}

// scheduleJob registers a gocron job and updates the in-memory map.
func (s *Service) scheduleJob(job *CronJob) (gocron.Job, error) {
	def, err := buildJobDefinition(job.Schedule)
	if err != nil {
		return nil, err
	}

	j, err := s.scheduler.NewJob(
		def,
		gocron.NewTask(s.makeTask(job.ID)),
		gocron.WithName(job.Name),
		gocron.WithEventListeners(
			gocron.AfterJobRuns(func(_ uuid.UUID, _ string) {
				s.onJobSuccess(job.ID)
			}),
			gocron.AfterJobRunsWithError(func(_ uuid.UUID, _ string, err error) {
				s.onJobError(job.ID, err)
			}),
		),
	)
	if err != nil {
		return nil, err
	}

	if s.jobs == nil {
		s.jobs = make(map[string]gocron.Job)
	}
	s.jobs[job.ID] = j

	if nextRun, err := j.NextRun(); err == nil && !nextRun.IsZero() {
		job.State.NextRunAtMs = nextRun.UnixMilli()
	}
	return j, nil
}

func (s *Service) makeTask(jobID string) func() {
	return func() {
		s.mu.RLock()
		var job *CronJob
		for i := range s.store.Jobs {
			if s.store.Jobs[i].ID == jobID {
				job = &s.store.Jobs[i]
				break
			}
		}
		s.mu.RUnlock()
		if job == nil {
			return
		}
		s.executeJob(job)
	}
}

func (s *Service) executeJob(job *CronJob) {
	startMs := nowMs()
	logger.Infof("[Cron] executing job '%s' (%s)", job.Name, job.ID)

	switch job.Payload.Kind {
	case "agent_turn":
		msg := model.InboundMessage{
			Channel: job.Payload.Channel,
			ChatID:  job.Payload.To,
			Content: job.Payload.Message,
		}
		if msg.Channel == "" {
			msg.Channel = "cron"
		}
		if msg.ChatID == "" {
			msg.ChatID = "default"
		}
		bus.GetService().GetMessageBus().PublishInbound(msg)
	default:
		logger.Warnf("[Cron] unknown payload kind '%s' for job '%s'", job.Payload.Kind, job.Name)
	}

	job.State.LastRunAtMs = startMs
	job.UpdatedAtMs = nowMs()

	if job.Schedule.Kind == ScheduleKindAt {
		if job.DeleteAfterRun {
			s.RemoveJob(job.ID)
		} else {
			job.Enabled = false
			job.State.NextRunAtMs = 0
		}
	} else {
		job.State.NextRunAtMs = computeNextRun(job.Schedule, nowMs())
	}
}

func (s *Service) onJobSuccess(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.store.Jobs {
		if s.store.Jobs[i].ID == jobID {
			s.store.Jobs[i].State.LastStatus = "ok"
			s.store.Jobs[i].State.LastError = nil
			s.saveStoreLocked()
			logger.Infof("[Cron] job '%s' completed", s.store.Jobs[i].Name)
			return
		}
	}
}

func (s *Service) onJobError(jobID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.store.Jobs {
		if s.store.Jobs[i].ID == jobID {
			s.store.Jobs[i].State.LastStatus = "error"
			errMsg := err.Error()
			s.store.Jobs[i].State.LastError = &errMsg
			s.saveStoreLocked()
			logger.Errorf("[Cron] job '%s' failed: %v", s.store.Jobs[i].Name, err)
			return
		}
	}
}

func (s *Service) loadStore() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.storePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warnf("[Cron] failed to read store: %v", err)
		}
		s.store = &CronStore{Version: 1, Jobs: []CronJob{}}
		return
	}

	var store CronStore
	if err := json.Unmarshal(data, &store); err != nil {
		logger.Warnf("[Cron] failed to parse store: %v", err)
		s.store = &CronStore{Version: 1, Jobs: []CronJob{}}
		return
	}
	s.store = &store
}

func (s *Service) saveStoreLocked() {
	if s.store == nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0755); err != nil {
		logger.Errorf("[Cron] mkdir store path fail: %v", err)
		return
	}
	data, err := json.MarshalIndent(s.store, "", "  ")
	if err != nil {
		logger.Errorf("[Cron] marshal store fail: %v", err)
		return
	}
	if err := os.WriteFile(s.storePath, data, 0644); err != nil {
		logger.Errorf("[Cron] write store fail: %v", err)
	}
}

func (s *Service) newID() string {
	b := make([]byte, 4)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (i * 8))
	}
	return fmt.Sprintf("%x", b)[:8]
}

func buildJobDefinition(schedule CronSchedule) (gocron.JobDefinition, error) {
	switch schedule.Kind {
	case ScheduleKindAt:
		if schedule.AtMs <= 0 {
			return nil, fmt.Errorf("invalid atMs")
		}
		return gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(time.UnixMilli(schedule.AtMs)),
		), nil
	case ScheduleKindEvery:
		if schedule.EveryMs <= 0 {
			return nil, fmt.Errorf("invalid everyMs")
		}
		return gocron.DurationJob(time.Duration(schedule.EveryMs) * time.Millisecond), nil
	case ScheduleKindCron:
		if schedule.Expr == "" {
			return nil, fmt.Errorf("empty cron expression")
		}
		return gocron.CronJob(schedule.Expr, false), nil
	default:
		return nil, fmt.Errorf("unsupported schedule kind: %s", schedule.Kind)
	}
}

func computeNextRun(schedule CronSchedule, nowMs int64) int64 {
	switch schedule.Kind {
	case ScheduleKindAt:
		if schedule.AtMs > nowMs {
			return schedule.AtMs
		}
		return 0
	case ScheduleKindEvery:
		if schedule.EveryMs <= 0 {
			return 0
		}
		return nowMs + schedule.EveryMs
	case ScheduleKindCron:
		loc := time.Local
		if schedule.Tz != "" {
			if l, err := time.LoadLocation(schedule.Tz); err == nil {
				loc = l
			}
		}
		c := gocron.NewDefaultCron(false)
		if err := c.IsValid(schedule.Expr, loc, time.Now()); err != nil {
			return 0
		}
		next := c.Next(time.Now().In(loc))
		return next.UnixMilli()
	default:
		return 0
	}
}

func filterJobs(jobs []CronJob, keep func(CronJob) bool) []CronJob {
	out := make([]CronJob, 0, len(jobs))
	for _, j := range jobs {
		if keep(j) {
			out = append(out, j)
		}
	}
	return out
}

func nowMs() int64 {
	return time.Now().UnixMilli()
}
