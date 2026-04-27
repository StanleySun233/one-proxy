package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/config"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/service"
)

type Scheduler struct {
	service  *service.ControlPlane
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func New(service *service.ControlPlane, cfg config.Config) *Scheduler {
	interval, err := time.ParseDuration(cfg.SchedulerInterval)
	if err != nil || interval <= 0 {
		interval = time.Minute
	}
	return &Scheduler{
		service:  service,
		interval: interval,
	}
}

func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run(ctx)
	}()
}

func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func (s *Scheduler) run(ctx context.Context) {
	s.tick()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Scheduler) tick() {
	if err := s.service.RunMaintenance(); err != nil {
		log.Printf("maintenance failed: %v", err)
	}
}
