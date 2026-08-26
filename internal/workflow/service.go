package workflow

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"encoding/json"
	"sync"
	"time"
)

type Service struct {
	store         *storage.Store
	locks         sync.Map
	createReplays createReplayCache
	now           func() time.Time
}

func New(store *storage.Store) *Service {
	return &Service{
		store:         store,
		createReplays: createReplayCache{responses: make(map[string][]byte)},
		now:           func() time.Time { return time.Now().UTC() },
	}
}

type createReplayCache struct {
	mu        sync.RWMutex
	responses map[string][]byte
}

func (c *createReplayCache) load(requestID string) (*domain.SampleBatch, bool, error) {
	c.mu.RLock()
	encoded, ok := c.responses[requestID]
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	var result domain.SampleBatch
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, false, err
	}
	return &result, true, nil
}

func (c *createReplayCache) store(requestID string, result *domain.SampleBatch) error {
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.responses[requestID] = encoded
	c.mu.Unlock()
	return nil
}

func (s *Service) withBatch(id string, fn func() error) error {
	value, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}

func (s *Service) Store() *storage.Store { return s.store }
