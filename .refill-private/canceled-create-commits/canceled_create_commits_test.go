package canceledcreatecommits

import (
	"anaerobic-release/internal/api"
	"anaerobic-release/internal/storage"
	"anaerobic-release/internal/workflow"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
)

type eofSignalBody struct {
	reader  *bytes.Reader
	reached chan struct{}
	once    sync.Once
}

func (b *eofSignalBody) Read(target []byte) (int, error) {
	n, err := b.reader.Read(target)
	if errors.Is(err, io.EOF) {
		b.once.Do(func() { close(b.reached) })
	}
	return n, err
}

func (b *eofSignalBody) Close() error { return nil }

func TestCanceledCreateCommitsAfterStorageWait(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}

	locked := make(chan struct{})
	unlock := make(chan struct{})
	blockerDone := make(chan error, 1)
	blockerErr := errors.New("release storage blocker")
	go func() {
		blockerDone <- store.Update(func(*storage.State) error {
			close(locked)
			<-unlock
			return blockerErr
		})
	}()
	<-locked

	body := &eofSignalBody{
		reader:  bytes.NewReader([]byte(`{"batch_id":"cancelled-batch","site_code":"SITE-CANCEL","stratum_reference":"L7","collector_id":"collector-cancel","collected_at":"2026-01-02T03:04:05Z","baseline_oxygen_ppm":20,"baseline_temperature_c":8,"expected_revision":0}`)),
		reached: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("POST", "/v1/batches", body).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "cancelled-create-request")
	recorder := httptest.NewRecorder()
	handlerDone := make(chan struct{})
	go func() {
		api.New(workflow.New(store)).Handler().ServeHTTP(recorder, req)
		close(handlerDone)
	}()

	<-body.reached
	cancel()
	close(unlock)
	if err := <-blockerDone; !errors.Is(err, blockerErr) {
		t.Fatalf("storage blocker returned %v", err)
	}
	<-handlerDone

	if recorder.Code < 500 {
		t.Fatalf("canceled request unexpectedly returned status %d", recorder.Code)
	}
	if _, err := store.Batch("cancelled-batch"); err == nil {
		t.Fatal("canceled request committed batch")
	}
}
