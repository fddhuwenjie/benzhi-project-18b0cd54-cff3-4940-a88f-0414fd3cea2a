package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Store) Update(fn func(*State) error) error {
	return s.UpdateContext(context.Background(), fn)
}

func (s *Store) UpdateContext(ctx context.Context, fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("请求已取消，事务未提交: %w", err)
	}
	copy, err := cloneState(s.state)
	if err != nil {
		return err
	}
	if err := fn(copy); err != nil {
		return err
	}
	if err := prepareRecovery(copy); err != nil {
		return fmt.Errorf("生成恢复元数据: %w", err)
	}
	if err := verifyState(copy, true); err != nil {
		return fmt.Errorf("提交前状态校验: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("请求已取消，事务未提交: %w", err)
	}
	if err := s.persist(copy); err != nil {
		return err
	}
	s.state = copy
	return nil
}

func (s *Store) View(fn func(*State) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fn(s.state)
}

func cloneState(source *State) (*State, error) {
	b, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var target State
	if err := json.Unmarshal(b, &target); err != nil {
		return nil, err
	}
	return &target, nil
}

func (s *Store) persist(next *State) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("创建数据目录: %w", err)
	}
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("编码快照: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时快照: %w", err)
	}
	tempName := temp.Name()
	ok := false
	defer func() {
		temp.Close()
		if !ok {
			os.Remove(tempName)
		}
	}()
	if err := temp.Chmod(0600); err != nil {
		return err
	}
	if _, err := temp.Write(b); err != nil {
		return fmt.Errorf("写入快照: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("同步快照: %w", err)
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, s.path); err != nil {
		return fmt.Errorf("原子替换快照: %w", err)
	}
	ok = true
	return nil
}
