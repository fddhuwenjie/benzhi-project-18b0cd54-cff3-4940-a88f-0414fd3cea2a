package api

import (
	"net/http"
)

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
	result, err := s.workflow.WithContext(r.Context()).CreateBatch(body.command(requestID(r)))
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
	result, err := s.workflow.WithContext(r.Context()).FreezePlan(body.command(r.PathValue("batch_id"), requestID(r)))
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, 200, result)
}
