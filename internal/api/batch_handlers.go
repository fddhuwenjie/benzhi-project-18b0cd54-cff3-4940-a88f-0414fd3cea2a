package api

import (
	"anaerobic-release/internal/domain"
	"context"
	"fmt"
	"net/http"
)

func finishWrite[T any](ctx context.Context, run func() (T, error)) (T, error) {
	result, err := run()
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, fmt.Errorf("请求已取消但写事务已经结束: %w", err)
	}
	return result, nil
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	head, err := s.workflow.Store().AuditHead()
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, 200, map[string]any{"status": "ok", "audit_head": head})
}

func (s *Server) CreateBatchHandler(w http.ResponseWriter, r *http.Request) {
	var body createBatchRequest
	if err := decode(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := finishWrite(r.Context(), func() (*domain.SampleBatch, error) {
		return s.workflow.CreateBatch(body.command(requestID(r)))
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, result)
}

func (s *Server) GetBatchHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.workflow.Batch(r.PathValue("batch_id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, 200, result)
}

func (s *Server) FreezePlanHandler(w http.ResponseWriter, r *http.Request) {
	var body freezePlanRequest
	if err := decode(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.workflow.FreezePlan(body.command(r.PathValue("batch_id"), requestID(r)))
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, 200, result)
}
