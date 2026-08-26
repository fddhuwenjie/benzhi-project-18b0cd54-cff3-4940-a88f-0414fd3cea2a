package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type selfCheckClient struct {
	base   string
	client *http.Client
}

func runSelfCheck(cfg config) error {
	temp, err := os.MkdirTemp("", "anaerobic-release-self-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	cfg.dataFile = filepath.Join(temp, "snapshot.json")
	server, err := buildServer(cfg)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", cfg.address)
	if err != nil {
		return fmt.Errorf("自检监听失败: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	client := selfCheckClient{base: "http://" + listener.Addr().String(), client: &http.Client{Timeout: 3 * time.Second}}
	checkErr := client.run()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	shutdownErr := server.Shutdown(ctx)
	serveErr := <-done
	if checkErr != nil {
		return checkErr
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	fmt.Println("自检通过：建档、冻结、转运、污染复核、放行、归档证据链均已完成")
	return nil
}

func (c selfCheckClient) run() error {
	digests := []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	}
	baseTime := time.Now().UTC()
	now := baseTime.Format(time.RFC3339Nano)
	checkpointAt := baseTime.Add(time.Minute).Format(time.RFC3339Nano)
	testedAt := baseTime.Add(2 * time.Minute).Format(time.RFC3339Nano)
	reviewedAt := baseTime.Add(3 * time.Minute).Format(time.RFC3339Nano)
	steps := []struct {
		method, path, key string
		body              map[string]any
		want              int
	}{
		{"GET", "/healthz", "", nil, 200},
		{"POST", "/v1/batches", "self-create", map[string]any{"batch_id": "SELF-CHECK-001", "site_code": "SITE-A", "stratum_reference": "L3", "collector_id": "collector-a", "collected_at": now, "baseline_oxygen_ppm": 20, "baseline_temperature_c": 8, "expected_revision": 0}, 201},
		{"POST", "/v1/batches/SELF-CHECK-001/preservation-plan", "self-plan", map[string]any{"expected_revision": 1, "handover_at": now, "handover_evidence_digest": digests[0], "container_id": "jar-1", "seal_method": "butyl-double-seal", "culture_target": "sulfate-reducers", "custodian_id": "transport-a", "max_oxygen_ppm": 100, "min_temperature_c": 2, "max_temperature_c": 12}, 200},
		{"POST", "/v1/batches/SELF-CHECK-001/checkpoints", "self-checkpoint", map[string]any{"expected_revision": 2, "recorded_by": "transport-a", "recorded_at": checkpointAt, "oxygen_ppm": 30, "temperature_c": 7, "seal_intact": true, "location_note": "实验室交接", "evidence_digest": digests[1]}, 201},
		{"POST", "/v1/batches/SELF-CHECK-001/contamination-tests", "self-test", map[string]any{"expected_revision": 3, "result": "not_detected", "tested_by": "lab-a", "tested_at": testedAt, "method": "blank-control", "evidence_digest": digests[2]}, 200},
		{"POST", "/v1/batches/SELF-CHECK-001/reviews", "self-review", map[string]any{"expected_revision": 4, "reviewer_id": "quality-a", "reviewed_at": reviewedAt, "decision": "approve", "evidence_digest": digests[3]}, 200},
		{"POST", "/v1/batches/SELF-CHECK-001/release", "self-release", map[string]any{"expected_revision": 5, "issuer_id": "issuer-a", "evidence_digest": digests[4]}, 201},
		{"GET", "/v1/batches/SELF-CHECK-001/evidence", "", nil, 200},
	}
	for _, step := range steps {
		if err := c.call(step.method, step.path, step.key, step.body, step.want); err != nil {
			return err
		}
	}
	return nil
}

func (c selfCheckClient) call(method, path, key string, body map[string]any, want int) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != want {
		return fmt.Errorf("自检 %s %s: 期望 %d，实际 %d: %s", method, path, want, resp.StatusCode, payload)
	}
	return nil
}
