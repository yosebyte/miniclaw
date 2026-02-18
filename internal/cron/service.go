// MIT License - Copyright (c) 2026 yosebyte
package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	robfig "github.com/robfig/cron/v3"
)

// Job is a single scheduled task.
type Job struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Schedule string    `json:"schedule"` // cron expression or "@every Xm"
	Message  string    `json:"message"`
	ChatID   string    `json:"chatId"`
	Enabled  bool      `json:"enabled"`
	Created  time.Time `json:"created"`
}

// SendFunc is called when a job fires to deliver its message.
type SendFunc func(chatID, text string) error

// Service manages scheduled cron jobs.
type Service struct {
	mu       sync.Mutex
	jobs     map[string]*Job
	entryIDs map[string]robfig.EntryID
	cron     *robfig.Cron
	path     string
	send     SendFunc
	agentFn  func(ctx context.Context, chatID, message string) (string, error)
}

// New creates and starts a CronService.
// agentFn is called when a job fires; its result is sent via sendFn.
func New(path string, send SendFunc, agentFn func(ctx context.Context, chatID, message string) (string, error)) *Service {
	s := &Service{
		jobs:     make(map[string]*Job),
		entryIDs: make(map[string]robfig.EntryID),
		cron:     robfig.New(robfig.WithSeconds()),
		path:     path,
		send:     send,
		agentFn:  agentFn,
	}
	_ = s.load()
	return s
}

// Start begins the scheduler.
func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range s.jobs {
		if job.Enabled {
			s.schedule(job)
		}
	}
	s.cron.Start()
	slog.Info("cron service started", "jobs", len(s.jobs))
}

// Stop halts the scheduler.
func (s *Service) Stop() {
	s.cron.Stop()
}

// Add creates a new job and schedules it.
func (s *Service) Add(name, schedule, message, chatID string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := &Job{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:     name,
		Schedule: schedule,
		Message:  message,
		ChatID:   chatID,
		Enabled:  true,
		Created:  time.Now().UTC(),
	}

	if err := s.schedule(job); err != nil {
		return nil, fmt.Errorf("invalid schedule %q: %w", schedule, err)
	}

	s.jobs[job.ID] = job
	_ = s.save()
	slog.Info("cron job added", "id", job.ID, "name", job.Name, "schedule", job.Schedule)
	return job, nil
}

// Remove deletes a job by ID.
func (s *Service) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job %q not found", id)
	}
	if entryID, ok := s.entryIDs[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entryIDs, id)
	}
	delete(s.jobs, id)
	_ = s.save()
	slog.Info("cron job removed", "id", id, "name", job.Name)
	return nil
}

// List returns all jobs sorted by creation time.
func (s *Service) List() []*Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	return jobs
}

// AddJob satisfies the agent.CronService interface (discards the returned job).
func (s *Service) AddJob(name, schedule, message, chatID string) error {
	_, err := s.Add(name, schedule, message, chatID)
	return err
}

// ListFormatted returns a human-readable list of all jobs.
func (s *Service) ListFormatted() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.jobs) == 0 {
		return "No cron jobs scheduled."
	}
	var sb strings.Builder
	for _, j := range s.jobs {
		status := "enabled"
		if !j.Enabled {
			status = "disabled"
		}
		fmt.Fprintf(&sb, "• [%s] %s — %s (%s)\n  Message: %s\n",
			j.ID, j.Name, j.Schedule, status, j.Message)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (s *Service) schedule(job *Job) error {
	entryID, err := s.cron.AddFunc(job.Schedule, func() {
		slog.Info("cron job firing", "id", job.ID, "name", job.Name)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		result, err := s.agentFn(ctx, job.ChatID, job.Message)
		if err != nil {
			slog.Error("cron agent error", "id", job.ID, "err", err)
			if s.send != nil {
				_ = s.send(job.ChatID, "⚠️ Cron job error: "+err.Error())
			}
			return
		}
		if s.send != nil && result != "" {
			_ = s.send(job.ChatID, result)
		}
	})
	if err != nil {
		return err
	}
	s.entryIDs[job.ID] = entryID
	return nil
}

func (s *Service) save() error {
	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *Service) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var jobs []*Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return err
	}
	for _, j := range jobs {
		s.jobs[j.ID] = j
	}
	return nil
}
