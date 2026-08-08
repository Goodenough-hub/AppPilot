package blog

import (
	"context"
	"log"
	"time"
)

// Scheduler 周期性扫描到点的定时草稿并提升为已发布。
// 由 server 启动时通过 Start 拉起，进程退出时通过 ctx 取消。
type Scheduler struct {
	repo   *Repository
	tick   time.Duration
	logger *log.Logger
}

func NewScheduler(repo *Repository) *Scheduler {
	return &Scheduler{
		repo:   repo,
		tick:   60 * time.Second,
		logger: log.Default(),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.tick)
	go func() {
		defer ticker.Stop()
		s.publishOnce() // 启动时立即跑一次，补齐停机期间到点的定时发布
		for {
			select {
			case <-ticker.C:
				s.publishOnce()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Scheduler) publishOnce() {
	ids, err := s.repo.PublishScheduledDrafts()
	if err != nil {
		s.logger.Printf("[blog-scheduler] publish due failed: %v", err)
		return
	}
	if len(ids) > 0 {
		s.logger.Printf("[blog-scheduler] published %d scheduled draft(s): %v", len(ids), ids)
	}
}
