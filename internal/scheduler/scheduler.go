package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/ternarybob/gitsync/internal/config"
	"github.com/ternarybob/gitsync/internal/logger"
	"github.com/ternarybob/gitsync/internal/store"
	gitsync "github.com/ternarybob/gitsync/internal/sync"
)

type Scheduler struct {
	cron   *cron.Cron
	jobs   map[string]cron.EntryID
	store  *store.Store
	config *config.Config
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func New(cfg *config.Config, s *store.Store) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		cron:   cron.New(cron.WithSeconds()),
		jobs:   make(map[string]cron.EntryID),
		store:  s,
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Scheduler) Start() error {
	logger.Info("Starting scheduler")

	for _, job := range s.config.Jobs {
		if !job.Enabled {
			logger.WithField("job", job.Name).Info("Job is disabled, skipping")
			continue
		}

		if err := s.scheduleJob(&job); err != nil {
			logger.WithFields(map[string]interface{}{
				"job":   job.Name,
				"error": err,
			}).Error("Failed to schedule job")
			continue
		}
	}

	s.cron.Start()

	go s.cleanupLoop()

	logger.Infof("Scheduler started with %d active jobs", len(s.jobs))
	return nil
}

func (s *Scheduler) Stop() {
	logger.Info("Stopping scheduler")

	s.cancel()

	ctx := s.cron.Stop()
	<-ctx.Done()

	logger.Info("Scheduler stopped")
}

func (s *Scheduler) scheduleJob(jobConfig *config.JobConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[jobConfig.Name]; exists {
		logger.WithField("job", jobConfig.Name).Warn("Job already scheduled")
		return nil
	}

	syncer, err := gitsync.NewSyncer(jobConfig, s.store)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}

	jobFunc := s.createJobFunc(jobConfig, syncer)

	entryID, err := s.cron.AddFunc(jobConfig.Schedule, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.jobs[jobConfig.Name] = entryID

	logger.WithFields(map[string]interface{}{
		"job":      jobConfig.Name,
		"schedule": jobConfig.Schedule,
	}).Info("Job scheduled successfully")

	return nil
}

func (s *Scheduler) createJobFunc(jobConfig *config.JobConfig, syncer *gitsync.Syncer) func() {
	return func() {
		logger.WithField("job", jobConfig.Name).Info("Executing scheduled job")

		ctx := s.ctx
		if jobConfig.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(s.ctx, jobConfig.Timeout)
			defer cancel()
		}

		startTime := time.Now()

		if err := syncer.SyncAll(ctx); err != nil {
			logger.WithFields(map[string]interface{}{
				"job":      jobConfig.Name,
				"error":    err,
				"duration": time.Since(startTime).Seconds(),
			}).Error("Job execution failed")
		} else {
			logger.WithFields(map[string]interface{}{
				"job":      jobConfig.Name,
				"duration": time.Since(startTime).Seconds(),
			}).Info("Job execution completed")
		}
	}
}

func (s *Scheduler) RunJobNow(jobName string) error {
	for _, job := range s.config.Jobs {
		if job.Name == jobName {
			syncer, err := gitsync.NewSyncer(&job, s.store)
			if err != nil {
				return fmt.Errorf("failed to create syncer: %w", err)
			}

			ctx := context.Background()
			if job.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, job.Timeout)
				defer cancel()
			}

			return syncer.SyncAll(ctx)
		}
	}
	return fmt.Errorf("job not found: %s", jobName)
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

	transactions, err := s.store.GetTransactionsByJob(jobName, 10)
	if err == nil {
		status["recent_transactions"] = transactions
	}

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

func (s *Scheduler) cleanupLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if s.config.Store.RetentionDays > 0 {
				logger.Info("Running transaction cleanup")
				if err := s.store.CleanupOldTransactions(s.config.Store.RetentionDays); err != nil {
					logger.WithField("error", err).Error("Failed to cleanup old transactions")
				}
			}
		}
	}
}
