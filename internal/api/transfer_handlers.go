package api

import (
	"anaerobic-release/internal/domain"
	"net/http"
)

func (s *Server) AddCheckpointHandler(w http.ResponseWriter, r *http.Request) {
	var body checkpointRequest
	if err := decode(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.workflow.WithContext(r.Context()).AddCheckpoint(body.command(r.PathValue("batch_id"), requestID(r)))
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, result)
}

func (s *Server) ResolveDeviationHandler(w http.ResponseWriter, r *http.Request) {
	var body resolveRequest
	if err := decode(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	batchID := r.PathValue("batch_id")
	retest := body.Retest.command(batchID, requestID(r))
	cmd := domain.ResolveDeviationCommand{WriteMeta: domain.WriteMeta{RequestID: requestID(r), ExpectedRevision: body.ExpectedRevision}, BatchID: batchID, DeviationID: r.PathValue("deviation_id"), CorrectiveAction: body.CorrectiveAction, ResolvedBy: body.ResolvedBy, ResolutionDigest: body.ResolutionDigest, Retest: retest}
	result, err := s.workflow.WithContext(r.Context()).ResolveDeviation(cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, 200, result)
}
