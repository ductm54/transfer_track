// Package scheduler provides scheduling functionality for periodic tasks.
package scheduler

import (
	"context"
	"time"

	"github.com/ductm54/transfer-track/internal/service"
	"go.uber.org/zap"
)

// Scheduler handles scheduled tasks.
type Scheduler struct {
	transferService *service.TransferService
	logger          *zap.SugaredLogger
	stopCh          chan struct{}
}

// NewScheduler creates a new Scheduler.
func NewScheduler(transferService *service.TransferService, logger *zap.SugaredLogger) *Scheduler {
	return &Scheduler{
		transferService: transferService,
		logger:          logger,
		stopCh:          make(chan struct{}),
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	go s.run()
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// run runs the scheduler.
func (s *Scheduler) run() {
	s.logger.Infow("Starting scheduler")

	// Run immediately on startup
	s.runDailyUpdate()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndRunDailyUpdate()
		case <-s.stopCh:
			s.logger.Infow("Stopping scheduler")
			return
		}
	}
}

// checkAndRunDailyUpdate checks if it's time to run the daily update.
func (s *Scheduler) checkAndRunDailyUpdate() {
	ctx := context.Background()

	// Get the configured daily refresh time
	timeStr, err := s.transferService.GetDailyRefreshTime(ctx)
	if err != nil {
		s.logger.Errorw("Error getting daily refresh time", "err", err)
		return
	}

	// Parse the time
	refreshTime, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		s.logger.Errorw("Error parsing daily refresh time", "err", err)
		return
	}

	// Get current time
	now := time.Now()
	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)
	scheduledTime := time.Date(0, 1, 1, refreshTime.Hour(), refreshTime.Minute(), 0, 0, time.UTC)

	// Check if it's time to run the daily update
	if currentTime.Equal(scheduledTime) {
		s.runDailyUpdate()
	}
}

// runDailyUpdate runs the daily update.
func (s *Scheduler) runDailyUpdate() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	s.logger.Infow("Running daily update")

	err := s.transferService.FetchAndStoreTransfers(ctx)
	if err != nil {
		s.logger.Errorw("Error running daily update", "err", err)
		return
	}

	s.logger.Infow("Daily update completed successfully")
}
