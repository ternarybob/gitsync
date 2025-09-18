package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/ternarybob/gitsync/internal/common"
)

type Scheduler struct {
	cron   *cron.Cron
	jobs   map[string]cron.EntryID
	config *common.Config
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func NewScheduler(cfg *common.Config) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		cron:   cron.New(cron.WithSeconds()),
		jobs:   make(map[string]cron.EntryID),
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Scheduler) Start() error {
	logger := common.GetLogger()
	logger.Info().Msg("Starting scheduler")

	for _, jobName := range s.config.Jobs.Names {
		jobConfig, exists := s.config.GetJobConfig(jobName)
		if !exists {
			logger.Error().Str("job", jobName).Msg("Job definition not found")
			continue
		}

		if !jobConfig.Enabled {
			logger.Info().Str("job", jobName).Msg("Job is disabled, skipping")
			continue
		}

		if err := s.scheduleJob(jobName, jobConfig); err != nil {
			logger.Error().Str("job", jobName).Err(err).Msg("Failed to schedule job")
			continue
		}
	}

	s.cron.Start()

	logger.Info().Int("active_jobs", len(s.jobs)).Msg("Scheduler started")
	return nil
}

func (s *Scheduler) Stop() {
	logger := common.GetLogger()
	logger.Info().Msg("Stopping scheduler")

	s.cancel()

	ctx := s.cron.Stop()
	<-ctx.Done()

	logger.Info().Msg("Scheduler stopped")
}

func (s *Scheduler) scheduleJob(jobName string, jobConfig *common.JobConfig) error {
	logger := common.GetLogger()
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[jobName]; exists {
		logger.Warn().Str("job", jobName).Msg("Job already scheduled")
		return nil
	}

	syncer, err := NewSyncer(jobName, jobConfig)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}

	jobFunc := s.createJobFunc(jobName, jobConfig, syncer)

	entryID, err := s.cron.AddFunc(s.config.Jobs.Schedule, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.jobs[jobName] = entryID

	logger.Info().Str("job", jobName).Str("schedule", s.config.Jobs.Schedule).Msg("Job scheduled successfully")

	return nil
}

func (s *Scheduler) createJobFunc(jobName string, jobConfig *common.JobConfig, syncer *Syncer) func() {
	return func() {
		logger := common.GetLogger()
		logger.Info().Str("job", jobName).Msg("Executing scheduled job")

		ctx := s.ctx
		if s.config.Jobs.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(s.ctx, s.config.Jobs.Timeout)
			defer cancel()
		}

		startTime := time.Now()

		if err := syncer.SyncAll(ctx); err != nil {
			logger.Error().Str("job", jobName).Err(err).Float64("duration", time.Since(startTime).Seconds()).Msg("Job execution failed")
		} else {
			logger.Info().Str("job", jobName).Float64("duration", time.Since(startTime).Seconds()).Msg("Job execution completed")
		}
	}
}

func (s *Scheduler) RunJobNow(jobName string) error {
	jobConfig, exists := s.config.GetJobConfig(jobName)
	if !exists {
		return fmt.Errorf("job not found: %s", jobName)
	}

	syncer, err := NewSyncer(jobName, jobConfig)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}

	ctx := context.Background()
	if s.config.Jobs.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.Jobs.Timeout)
		defer cancel()
	}

	return syncer.SyncAll(ctx)
}

func (s *Scheduler) GetJobStatus(jobName string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]interface{})

	entryID, exists := s.jobs[jobName]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobName)
	}

	entry := s.cron.Entry(entryID)
	status["job_name"] = jobName
	status["next_run"] = entry.Next
	status["prev_run"] = entry.Prev

	return status, nil
}

func (s *Scheduler) GetAllJobsStatus() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var statuses []map[string]interface{}

	for jobName, entryID := range s.jobs {
		entry := s.cron.Entry(entryID)
		status := map[string]interface{}{
			"job_name": jobName,
			"next_run": entry.Next,
			"prev_run": entry.Prev,
		}
		statuses = append(statuses, status)
	}

	return statuses
}
