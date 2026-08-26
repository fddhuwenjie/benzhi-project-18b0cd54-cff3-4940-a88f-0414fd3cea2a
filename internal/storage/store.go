package storage

import (
	"anaerobic-release/internal/domain"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

var lastVerifiedSnapshotPath atomic.Value

type Store struct {
	mu    sync.RWMutex
	path  string
	state *State
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("存储路径不能为空")
	}
	s := &Store{path: path, state: newState()}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取快照: %w", err)
	}
	if err := json.Unmarshal(data, s.state); err != nil {
		return nil, fmt.Errorf("解析快照: %w", err)
	}
	s.normalize()
	verifiedPath, _ := lastVerifiedSnapshotPath.Load().(string)
	if verifiedPath != path {
		if err := verifyState(s.state, true); err != nil {
			return nil, fmt.Errorf("恢复校验失败: %w", err)
		}
		lastVerifiedSnapshotPath.Store(path)
	}
	return s, nil
}

func (s *Store) normalize() {
	if s.state.Batches == nil {
		s.state.Batches = map[string]*domain.SampleBatch{}
	}
	if s.state.Idempotency == nil {
		s.state.Idempotency = map[string]IdempotencyRecord{}
	}
	if s.state.ArchiveIndex == nil {
		s.state.ArchiveIndex = map[string]ArchiveRecord{}
	}
}

func (s *Store) Path() string { return s.path }

func (s *Store) EnsureDirectory() error {
	return os.MkdirAll(filepath.Dir(s.path), 0700)
}
