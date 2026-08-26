package workflow

import (
	"anaerobic-release/internal/storage"
	"sync"
	"time"
)

type Service struct {
	store         *storage.Store
	locks         sync.Map
	evidenceCache sync.Map
	now           func() time.Time
}

func New(store *storage.Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) withBatch(id string, fn func() error) error {
	value, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}

func (s *Service) Store() *storage.Store { return s.store }
