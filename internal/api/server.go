package api

import (
	"anaerobic-release/internal/workflow"
	"net/http"
)

type Server struct {
	workflow *workflow.Service
	mux      *http.ServeMux
}

func New(service *workflow.Service) *Server {
	s := &Server{workflow: service, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return requestMiddleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.HealthHandler)
	s.mux.HandleFunc("POST /v1/batches", s.CreateBatchHandler)
	s.mux.HandleFunc("GET /v1/batches/{batch_id}", s.GetBatchHandler)
	s.mux.HandleFunc("POST /v1/batches/{batch_id}/preservation-plan", s.FreezePlanHandler)
	s.mux.HandleFunc("POST /v1/batches/{batch_id}/checkpoints", s.AddCheckpointHandler)
	s.mux.HandleFunc("POST /v1/batches/{batch_id}/deviations/{deviation_id}/resolve", s.ResolveDeviationHandler)
	s.mux.HandleFunc("POST /v1/batches/{batch_id}/contamination-tests", s.ContaminationHandler)
	s.mux.HandleFunc("POST /v1/batches/{batch_id}/reviews", s.ReviewHandler)
	s.mux.HandleFunc("POST /v1/batches/{batch_id}/release", s.ReleaseHandler)
	s.mux.HandleFunc("GET /v1/batches/{batch_id}/evidence", s.EvidenceHandler)
}
