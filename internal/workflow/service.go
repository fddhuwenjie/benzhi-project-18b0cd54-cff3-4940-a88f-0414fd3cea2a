package workflow

import (
	"anaerobic-release/internal/storage"
	"context"
	"fmt"
	"sync"
	"time"
)

type Service struct {
	store *storage.Store
	locks *sync.Map
	now   func() time.Time
	ctx   context.Context
}

func New(store *storage.Store) *Service {
	return &Service{store: store, locks: &sync.Map{}, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	return &Service{store: s.store, locks: s.locks, now: s.now, ctx: ctx}
}

func (s *Service) context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *Service) withBatch(id string, fn func() error) error {
	value, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	if err := s.context().Err(); err != nil {
		return fmt.Errorf("请求已取消，事务未提交: %w", err)
	}
	return fn()
}

func (s *Service) Store() *storage.Store { return s.store }
