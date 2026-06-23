package scheduler

import (
	"log/slog"
	"sync"
	"time"
)

type Scheduler struct {
	interval time.Duration
	ticker   *time.Ticker
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
	onTick   func()
}

func NewScheduler(interval time.Duration) *Scheduler {
	return &Scheduler{interval: interval}
}

func (s *Scheduler) Start(onTick func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	s.onTick = onTick
	s.ticker = time.NewTicker(s.interval)
	s.stopCh = make(chan struct{})
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		slog.Info("scheduler started", "interval", s.interval)
		s.runTick()
		for {
			select {
			case <-s.ticker.C:
				s.runTick()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	s.ticker.Stop()
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

func (s *Scheduler) runTick() {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("scheduler panic", "error", rec)
		}
	}()
	if s.onTick != nil {
		s.onTick()
	}
}