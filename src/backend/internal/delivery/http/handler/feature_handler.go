package handler

import (
	"devtool/internal/delivery/http/middleware"
	"devtool/internal/domain"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const featurePolicyRequestMaxBytes int64 = 4096

type FeatureHandler struct {
	features domain.FeatureService
	validate *validator.Validate
}

func NewFeatureHandler(features domain.FeatureService) *FeatureHandler {
	return &FeatureHandler{features: features, validate: validator.New()}
}

type featurePolicyUpdateRequest struct {
	RequestID                 string `json:"requestId" validate:"required,uuid"`
	ExpectedRevision          string `json:"expectedRevision" validate:"required"`
	MemberComments            *bool  `json:"memberComments,omitempty"`
	MemberHistory             *bool  `json:"memberHistory,omitempty"`
	CommentsEnabled           *bool  `json:"commentsEnabled,omitempty"`
	ActivityEnabled           *bool  `json:"activityEnabled,omitempty"`
	VersionsEnabled           *bool  `json:"versionsEnabled,omitempty"`
	LibrariesEnabled          *bool  `json:"librariesEnabled,omitempty"`
	CollectionsEnabled        *bool  `json:"collectionsEnabled,omitempty"`
	PresentationEnabled       *bool  `json:"presentationEnabled,omitempty"`
	ScreenShareEnabled        *bool  `json:"screenShareEnabled,omitempty"`
	EnterpriseIdentityEnabled *bool  `json:"enterpriseIdentityEnabled,omitempty"`
	RetentionEnabled          *bool  `json:"retentionEnabled,omitempty"`
}

type featurePolicyResponse struct {
	WorkspaceID               uuid.UUID `json:"workspaceId"`
	Revision                  string    `json:"revision"`
	MemberComments            bool      `json:"memberComments"`
	MemberHistory             bool      `json:"memberHistory"`
	CommentsEnabled           bool      `json:"commentsEnabled"`
	ActivityEnabled           bool      `json:"activityEnabled"`
	VersionsEnabled           bool      `json:"versionsEnabled"`
	LibrariesEnabled          bool      `json:"librariesEnabled"`
	CollectionsEnabled        bool      `json:"collectionsEnabled"`
	PresentationEnabled       bool      `json:"presentationEnabled"`
	ScreenShareEnabled        bool      `json:"screenShareEnabled"`
	EnterpriseIdentityEnabled bool      `json:"enterpriseIdentityEnabled"`
	RetentionEnabled          bool      `json:"retentionEnabled"`
}

func featurePolicyPayload(policy domain.WorkspaceFeaturePolicy) featurePolicyResponse {
	return featurePolicyResponse{
		WorkspaceID: policy.WorkspaceID, Revision: strconv.FormatInt(policy.Revision, 10),
		MemberComments: policy.MemberComments, MemberHistory: policy.MemberHistory,
		CommentsEnabled: policy.CommentsEnabled, ActivityEnabled: policy.ActivityEnabled,
		VersionsEnabled: policy.VersionsEnabled, LibrariesEnabled: policy.LibrariesEnabled,
		CollectionsEnabled: policy.CollectionsEnabled, PresentationEnabled: policy.PresentationEnabled,
		ScreenShareEnabled:        policy.ScreenShareEnabled,
		EnterpriseIdentityEnabled: policy.EnterpriseIdentityEnabled,
		RetentionEnabled:          policy.RetentionEnabled,
	}
}

func writeFeatureError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, domain.ErrRequestIDReused) {
		writeJSONResponse(w, http.StatusConflict, map[string]string{"code": "REQUEST_ID_REUSED", "message": "Request ID was already used for a different mutation", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	if errors.Is(err, domain.ErrRevisionConflict) {
		writeJSONResponse(w, http.StatusConflict, map[string]string{"code": "REVISION_CONFLICT", "message": "Feature policy changed; reload and retry", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	if errors.Is(err, domain.ErrFeatureDisabled) {
		writeJSONResponse(w, http.StatusForbidden, map[string]string{"code": "FEATURE_DISABLED", "message": "Feature is disabled by workspace policy", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	if errors.Is(err, domain.ErrInvalidFeatureInput) {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid feature request", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	middleware.WriteAccessError(w, r, err)
}

func (h *FeatureHandler) Get(w http.ResponseWriter, r *http.Request) {
	workspaceID, workspaceOK := middleware.WorkspaceIDFromContext(r.Context())
	userID, userOK := middleware.UserIDFromContext(r.Context())
	if !workspaceOK || !userOK {
		middleware.WriteAccessError(w, r, domain.ErrInaccessible)
		return
	}
	policy, err := h.features.Policy(r.Context(), domain.UserPrincipal(userID), workspaceID)
	if err != nil {
		writeFeatureError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, featurePolicyPayload(policy))
}

func (h *FeatureHandler) Update(w http.ResponseWriter, r *http.Request) {
	workspaceID, workspaceOK := middleware.WorkspaceIDFromContext(r.Context())
	userID, userOK := middleware.UserIDFromContext(r.Context())
	if !workspaceOK || !userOK {
		middleware.WriteAccessError(w, r, domain.ErrInaccessible)
		return
	}
	var input featurePolicyUpdateRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, featurePolicyRequestMaxBytes))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&input)
	if err == nil {
		var extra any
		err = decoder.Decode(&extra)
		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = domain.ErrInvalidShareInput
		}
	}
	requestID, requestErr := uuid.Parse(input.RequestID)
	expectedRevision, revisionErr := strconv.ParseInt(input.ExpectedRevision, 10, 64)
	patch := domain.WorkspaceFeaturePolicyPatch{
		MemberComments: input.MemberComments, MemberHistory: input.MemberHistory,
		CommentsEnabled: input.CommentsEnabled, ActivityEnabled: input.ActivityEnabled,
		VersionsEnabled: input.VersionsEnabled, LibrariesEnabled: input.LibrariesEnabled,
		CollectionsEnabled: input.CollectionsEnabled, PresentationEnabled: input.PresentationEnabled,
		ScreenShareEnabled:        input.ScreenShareEnabled,
		EnterpriseIdentityEnabled: input.EnterpriseIdentityEnabled,
		RetentionEnabled:          input.RetentionEnabled,
	}
	if err != nil || requestErr != nil || revisionErr != nil || expectedRevision < 1 ||
		strconv.FormatInt(expectedRevision, 10) != input.ExpectedRevision ||
		h.validate.Struct(input) != nil || patch.Empty() {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"code": "INVALID_INPUT", "message": "Invalid feature policy request", "requestId": chimiddleware.GetReqID(r.Context())})
		return
	}
	policy, err := h.features.UpdatePolicy(r.Context(), domain.UserPrincipal(userID), workspaceID, expectedRevision, requestID, patch)
	if err != nil {
		writeFeatureError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSONResponse(w, http.StatusOK, featurePolicyPayload(policy))
}
