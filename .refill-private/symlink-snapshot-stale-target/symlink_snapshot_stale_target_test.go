package symlink_snapshot_stale_target_test

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/workflow"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSymlinkRetargetPreservesAcknowledgedCommit(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "volume-a")
	second := filepath.Join(root, "volume-b")
	if err := os.Mkdir(first, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(second, 0700); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(root, "active-volume")
	if err := os.Symlink(first, active); err != nil {
		t.Fatal(err)
	}

	snapshot := filepath.Join(active, "snapshot.json")
	store, err := storage.Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.New(store)
	collectedAt := time.Now().UTC().Add(-time.Minute)
	created, err := service.CreateBatch(domain.CreateBatchCommand{
		WriteMeta:            domain.WriteMeta{RequestID: "create-before-retarget"},
		BatchID:              "SYMLINK-BATCH-001",
		SiteCode:             "SITE-SYMLINK",
		StratumReference:     "L7",
		CollectorID:          "collector-symlink",
		CollectedAt:          collectedAt,
		BaselineOxygenPPM:    20,
		BaselineTemperatureC: 8,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(active); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(second, active); err != nil {
		t.Fatal(err)
	}

	_, err = service.FreezePlan(domain.FreezePlanCommand{
		WriteMeta: domain.WriteMeta{
			RequestID:        "freeze-after-retarget",
			ExpectedRevision: created.Revision,
		},
		BatchID: "SYMLINK-BATCH-001",
		Plan: domain.PreservationPlan{
			ContainerID:     "jar-symlink",
			SealMethod:      "butyl-double-seal",
			CultureTarget:   "sulfate-reducers",
			CustodianID:     "custodian-symlink",
			MaxOxygenPPM:    100,
			MinTemperatureC: 2,
			MaxTemperatureC: 12,
		},
		HandoverAt:             collectedAt,
		HandoverEvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("freeze should be acknowledged after retarget: %v", err)
	}

	restarted, err := storage.Open(snapshot)
	if err != nil {
		t.Fatalf("restart from configured snapshot path failed: %v", err)
	}
	batch, err := restarted.Batch("SYMLINK-BATCH-001")
	if err != nil || batch.Status != domain.StatusReadyTransfer || batch.Revision != 2 {
		t.Fatalf("retargeted snapshot lost an acknowledged plan: batch=%+v err=%v", batch, err)
	}
}
