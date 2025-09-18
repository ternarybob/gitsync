package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/ternarybob/gitsync/internal/common"
	gitsync "github.com/ternarybob/gitsync/internal/sync"
)

type Scheduler struct {
	cron   *cron.Cron
	jobs   map[string]cron.EntryID
	config *common.Config
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func New(cfg *common.Config) *Scheduler {
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
	common.Info("Starting scheduler")

	for _, jobName := range s.config.Jobs.Names {
		jobConfig, exists := s.config.GetJobConfig(jobName)
		if !exists {
			common.WithField("job", jobName).Error("Job definition not found")
			continue
		}

		if !jobConfig.Enabled {
			common.WithField("job", jobName).Info("Job is disabled, skipping")
			continue
		}

		if err := s.scheduleJob(jobName, jobConfig); err != nil {
			common.WithFields(map[string]interface{}{
				"job":   jobName,
				"error": err,
			}).Error("Failed to schedule job")
			continue
		}
	}

	s.cron.Start()

	common.Infof("Scheduler started with %d active jobs", len(s.jobs))
	return nil
}

func (s *Scheduler) Stop() {
	common.Info("Stopping scheduler")

	s.cancel()

	ctx := s.cron.Stop()
	<-ctx.Done()

	common.Info("Scheduler stopped")
}

func (s *Scheduler) scheduleJob(jobName string, jobConfig *common.JobConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[jobName]; exists {
		common.WithField("job", jobName).Warn("Job already scheduled")
		return nil
	}

	syncer, err := gitsync.NewSyncer(jobName, jobConfig)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}

	jobFunc := s.createJobFunc(jobName, jobConfig, syncer)

	entryID, err := s.cron.AddFunc(s.config.Jobs.Schedule, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.jobs[jobName] = entryID

	common.WithFields(map[string]interface{}{
		"job":      jobName,
		"schedule": s.config.Jobs.Schedule,
	}).Info("Job scheduled successfully")

	return nil
}

func (s *Scheduler) createJobFunc(jobName string, jobConfig *common.JobConfig, syncer *gitsync.Syncer) func() {
	return func() {
		common.WithField("job", jobName).Info("Executing scheduled job")

		ctx := s.ctx
		if s.config.Jobs.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(s.ctx, s.config.Jobs.Timeout)
			defer cancel()
		}

		startTime := time.Now()

		if err := syncer.SyncAll(ctx); err != nil {
			common.WithFields(map[string]interface{}{
				"job":      jobName,
				"error":    err,
				"duration": time.Since(startTime).Seconds(),
			}).Error("Job execution failed")
		} else {
			common.WithFields(map[string]interface{}{
				"job":      jobName,
				"duration": time.Since(startTime).Seconds(),
			}).Info("Job execution completed")
		}
	}
}

func (s *Scheduler) RunJobNow(jobName string) error {
	jobConfig, exists := s.config.GetJobConfig(jobName)
	if !exists {
		return fmt.Errorf("job not found: %s", jobName)
	}

	syncer, err := gitsync.NewSyncer(jobName, jobConfig)
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