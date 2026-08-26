package storage

import (
	"anaerobic-release/internal/domain"
	"path/filepath"
	"testing"
	"time"
)

func TestAtomicSnapshotCanRecover(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "snapshot.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Update(func(state *State) error {
		batch := &domain.SampleBatch{BatchID: "b1", Status: domain.StatusDraft, Revision: 1}
		state.Batches[batch.BatchID] = batch
		_, err := state.Audit.Append("b1", "batch.created", 1, time.Now().UTC(), batch)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := reopened.Batch("b1")
	if err != nil {
		t.Fatal(err)
	}
	if batch.Revision != 1 {
		t.Fatalf("revision=%d", batch.Revision)
	}
}

func TestUpdateRollsBackOnError(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := domain.NewError("test", "rollback", 409)
	err = store.Update(func(state *State) error { state.Batches["bad"] = &domain.SampleBatch{BatchID: "bad"}; return want })
	if err == nil {
		t.Fatal("应返回错误")
	}
	if _, err := store.Batch("bad"); err == nil {
		t.Fatal("失败事务不应写入")
	}
}
