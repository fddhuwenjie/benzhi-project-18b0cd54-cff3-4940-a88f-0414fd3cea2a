package stale_open_verification_cache_test

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenRevalidatesReplacedSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	store, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	if err := store.Update(func(state *storage.State) error {
		batch := &domain.SampleBatch{
			BatchID:              "cache-recovery-batch",
			SiteCode:             "SITE-ORIGINAL",
			StratumReference:     "L7",
			CollectorID:          "collector-one",
			CollectedAt:          at,
			BaselineOxygenPPM:    20,
			BaselineTemperatureC: 8,
			Status:               domain.StatusDraft,
			Revision:             1,
			CreatedAt:            at,
		}
		state.Batches[batch.BatchID] = batch
		_, err := state.Audit.Append(batch.BatchID, "batch.created", batch.Revision, at, batch)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := storage.Open(path); err != nil {
		t.Fatalf("合法快照首次恢复失败: %v", err)
	}

	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var replaced storage.State
	if err := json.Unmarshal(encoded, &replaced); err != nil {
		t.Fatal(err)
	}
	replaced.Batches["cache-recovery-batch"].SiteCode = "SITE-TAMPERED"
	encoded, err = json.MarshalIndent(&replaced, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}

	reopened, err := storage.Open(path)
	if err != nil {
		return
	}
	batch, readErr := reopened.Batch("cache-recovery-batch")
	if readErr != nil {
		t.Fatal(readErr)
	}
	t.Fatalf("TestOpenRevalidatesReplacedSnapshot: 被替换快照绕过恢复校验并返回 site_code=%s", batch.SiteCode)
}
