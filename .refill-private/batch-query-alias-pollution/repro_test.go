package batchqueryaliaspollution

import (
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/workflow"
	"path/filepath"
	"testing"
	"time"
)

func TestBatchQueryAliasPollutesPersistedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	store, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.New(store)
	collectedAt := time.Now().UTC().Add(-time.Hour)
	handoverAt := time.Now().UTC()

	create := func(batchID, requestID, siteCode string) {
		t.Helper()
		_, err := service.CreateBatch(domain.CreateBatchCommand{
			WriteMeta:            domain.WriteMeta{RequestID: requestID},
			BatchID:              batchID,
			SiteCode:             siteCode,
			StratumReference:     "layer-1",
			CollectorID:          "collector-1",
			CollectedAt:          collectedAt,
			BaselineOxygenPPM:    20,
			BaselineTemperatureC: 8,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	create("batch-original", "create-original", "site-original")
	frozen, err := service.FreezePlan(domain.FreezePlanCommand{
		WriteMeta:              domain.WriteMeta{RequestID: "freeze-original", ExpectedRevision: 1},
		BatchID:                "batch-original",
		HandoverAt:             handoverAt,
		HandoverEvidenceDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Plan: domain.PreservationPlan{
			ContainerID:     "container-1",
			SealMethod:      "double-seal",
			CultureTarget:   "anaerobic-culture",
			CustodianID:     "custodian-1",
			MaxOxygenPPM:    100,
			MinTemperatureC: 2,
			MaxTemperatureC: 12,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddCheckpoint(domain.AddCheckpointCommand{
		WriteMeta:      domain.WriteMeta{RequestID: "checkpoint-original", ExpectedRevision: frozen.Revision},
		BatchID:        "batch-original",
		CheckpointID:   "checkpoint-original",
		RecordedBy:     "custodian-1",
		RecordedAt:     handoverAt.Add(time.Minute),
		OxygenPPM:      20,
		TemperatureC:   8,
		SealIntact:     true,
		LocationNote:   "location-original",
		EvidenceDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	if err != nil {
		t.Fatal(err)
	}

	queried, err := service.Batch("batch-original")
	if err != nil {
		t.Fatal(err)
	}
	queried.Checkpoints[0].LocationNote = "location-tampered"

	create("batch-unrelated", "create-unrelated", "site-unrelated")

	reopened, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := reopened.Batch("batch-original")
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Checkpoints[0].LocationNote != "location-original" {
		t.Fatalf("query result mutation leaked into persisted state: location_note=%q", recovered.Checkpoints[0].LocationNote)
	}
}
