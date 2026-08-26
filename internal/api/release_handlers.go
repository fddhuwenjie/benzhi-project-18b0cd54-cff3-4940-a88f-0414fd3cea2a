package api

import (
	"anaerobic-release/internal/domain"
	"net/http"
)

func (s *Server) ContaminationHandler(w http.ResponseWriter, r *http.Request) {
	var body contaminationRequest
	if err := decode(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	cmd := domain.RecordContaminationCommand{WriteMeta: domain.WriteMeta{RequestID: requestID(r), ExpectedRevision: body.ExpectedRevision}, BatchID: r.PathValue("batch_id"), Result: body.Result, TestedBy: body.TestedBy, TestedAt: body.TestedAt, Method: body.Method, EvidenceDigest: body.EvidenceDigest}
	result, err := s.workflow.RecordContamination(cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, 200, result)
}

func (s *Server) ReviewHandler(w http.ResponseWriter, r *http.Request) {
	var body reviewRequest
	if err := decode(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	cmd := domain.ReviewCommand{WriteMeta: domain.WriteMeta{RequestID: requestID(r), ExpectedRevision: body.ExpectedRevision}, BatchID: r.PathValue("batch_id"), ReviewerID: body.ReviewerID, ReviewedAt: body.ReviewedAt, Decision: body.Decision, EvidenceDigest: body.EvidenceDigest}
	result, err := s.workflow.Review(cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, 200, result)
}

func (s *Server) ReleaseHandler(w http.ResponseWriter, r *http.Request) {
	var body releaseRequest
	if err := decode(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	cmd := domain.ReleaseCommand{WriteMeta: domain.WriteMeta{RequestID: requestID(r), ExpectedRevision: body.ExpectedRevision}, BatchID: r.PathValue("batch_id"), IssuerID: body.IssuerID, CertificateID: body.CertificateID, EvidenceDigest: body.EvidenceDigest}
	result, err := s.workflow.Release(cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, result)
}

func (s *Server) EvidenceHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.workflow.Evidence(r.PathValue("batch_id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	respond(w, 200, result)
}
