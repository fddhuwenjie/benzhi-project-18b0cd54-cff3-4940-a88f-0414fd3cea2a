package evidence_cache_cross_batch_test

import (
	"anaerobic-release/internal/api"
	"anaerobic-release/internal/domain"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/workflow"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestEvidenceCacheSeparatesArchivedBatches(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.New(store)
	base := time.Now().UTC().Add(-time.Hour)
	archiveBatch(t, service, "batch-alpha", base)
	archiveBatch(t, service, "batch-bravo", base.Add(10*time.Minute))

	handler := api.New(service).Handler()
	alpha := getEvidence(t, handler, "batch-alpha")
	if alpha.Package.BatchID != "batch-alpha" {
		t.Fatalf("alpha evidence batch_id=%q", alpha.Package.BatchID)
	}
	bravo := getEvidence(t, handler, "batch-bravo")
	if bravo.Package.BatchID != "batch-bravo" || bravo.Certificate == nil || bravo.Certificate.BatchID != "batch-bravo" {
		t.Fatalf("evidence cache returned another archived batch: package=%q certificate=%v", bravo.Package.BatchID, bravo.Certificate)
	}
}

func archiveBatch(t *testing.T, service *workflow.Service, batchID string, base time.Time) {
	t.Helper()
	request := func(step string) string { return batchID + "-" + step }
	batch, err := service.CreateBatch(domain.CreateBatchCommand{
		WriteMeta: domain.WriteMeta{RequestID: request("create")}, BatchID: batchID,
		SiteCode: "SITE", StratumReference: "L1", CollectorID: batchID + "-collector",
		CollectedAt: base, BaselineOxygenPPM: 20, BaselineTemperatureC: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.FreezePlan(domain.FreezePlanCommand{
		WriteMeta: domain.WriteMeta{RequestID: request("plan"), ExpectedRevision: batch.Revision}, BatchID: batchID,
		HandoverAt: base.Add(time.Minute), HandoverEvidenceDigest: digest(request("handover")),
		Plan: domain.PreservationPlan{ContainerID: batchID + "-jar", SealMethod: "double", CultureTarget: "anaerobic", CustodianID: batchID + "-carrier", MaxOxygenPPM: 100, MinTemperatureC: 2, MaxTemperatureC: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	eventBase := time.Now().UTC().Add(time.Minute)
	batch, err = service.AddCheckpoint(domain.AddCheckpointCommand{
		WriteMeta: domain.WriteMeta{RequestID: request("checkpoint"), ExpectedRevision: batch.Revision}, BatchID: batchID,
		RecordedBy: batchID + "-carrier", RecordedAt: eventBase, OxygenPPM: 20, TemperatureC: 8,
		SealIntact: true, LocationNote: "laboratory", EvidenceDigest: digest(request("checkpoint-evidence")),
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.RecordContamination(domain.RecordContaminationCommand{
		WriteMeta: domain.WriteMeta{RequestID: request("contamination"), ExpectedRevision: batch.Revision}, BatchID: batchID,
		Result: "not_detected", TestedBy: batchID + "-tester", TestedAt: eventBase.Add(time.Minute),
		Method: "blank-culture", EvidenceDigest: digest(request("contamination-evidence")),
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err = service.Review(domain.ReviewCommand{
		WriteMeta: domain.WriteMeta{RequestID: request("review"), ExpectedRevision: batch.Revision}, BatchID: batchID,
		ReviewerID: batchID + "-reviewer", ReviewedAt: eventBase.Add(2 * time.Minute), Decision: "approve",
		EvidenceDigest: digest(request("review-evidence")),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Release(domain.ReleaseCommand{
		WriteMeta: domain.WriteMeta{RequestID: request("release"), ExpectedRevision: batch.Revision}, BatchID: batchID,
		IssuerID: batchID + "-issuer", EvidenceDigest: digest(request("release-evidence")),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func getEvidence(t *testing.T, handler http.Handler, batchID string) workflow.EvidenceResult {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/batches/"+batchID+"/evidence", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET evidence status=%d body=%s", response.Code, response.Body.String())
	}
	var result workflow.EvidenceResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
