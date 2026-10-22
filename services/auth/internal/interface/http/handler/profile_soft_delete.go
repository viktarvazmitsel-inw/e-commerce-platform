package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/interface/http/response"
	"authorization/internal/usecase"
)

type ProfileSoftDeleteUseCase interface {
	Execute(ctx context.Context, input usecase.SoftDeleteProfileInput) (usecase.SoftDeleteProfileResult, error)
}

type ProfileSoftDeleteInput struct {
	TargetID string `json:"target_id"`
	Password string `json:"password"`
}

type ProfileSoftDeleteHandler struct {
	deleteProfile ProfileSoftDeleteUseCase
}

func NewProfileSoftDeleteHandler(uc ProfileSoftDeleteUseCase) *ProfileSoftDeleteHandler {
	return &ProfileSoftDeleteHandler{
		deleteProfile: uc,
	}
}

func (h *ProfileSoftDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		response.RespondJSON(w, http.StatusMethodNotAllowed, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	authorID, err := getUserIDFromContext(r.Context())
	if err != nil {
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_PARAMETERS",
				Message: err.Error(),
			},
		})
		return
	}

	var inputData ProfileSoftDeleteInput
	if !DecodeJSON(w, r, &inputData) {
		return
	}

	errorsDetails := validateInputData(inputData)
	if len(errorsDetails) > 0 {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: "invalid input data",
				Details: errorsDetails,
			},
		})
		return
	}

	softDeleteProfileInput := usecase.SoftDeleteProfileInput{
		AuthorID: strings.TrimSpace(authorID),
		TargetID: strings.TrimSpace(inputData.TargetID),
		Password: inputData.Password,
	}

	deletionResult, err := h.deleteProfile.Execute(r.Context(), softDeleteProfileInput)
	if err != nil {
		h.handleUsecaseError(w, err)
		return
	}

	if !deletionResult.SessionRevoked {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "SESSION_HAS_NOT_BEEN_REVOKED",
				Message: "session has not been revoked",
				Details: map[string]string{
					"target_id": inputData.TargetID,
				},
			},
		})
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
		Success: true,
		Data: map[string]any{
			"message":         "deletion successful",
			"session_revoked": deletionResult.SessionRevoked,
		},
	})
}

func validateInputData(inputData ProfileSoftDeleteInput) map[string]string {
	errorsDetails := make(map[string]string)

	trimmedTargetID := strings.TrimSpace(inputData.TargetID)
	if trimmedTargetID == "" {
		errorsDetails["user"] = "target user must be existing user"
	}

	if len(inputData.Password) < 8 || len(inputData.Password) > 50 {
		errorsDetails["password"] = "incorrect password"
	}

	return errorsDetails
}

func (h *ProfileSoftDeleteHandler) handleUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserIdNotFound):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
		response.RespondJSON(w, http.StatusForbidden, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "DEACTIVATED_USER",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrInvalidPassword):
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrUnauthorizedAction):
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		response.RespondJSON(w, http.StatusNotFound, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "TARGET_USER_NOT_FOUND",
				Message: err.Error(),
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "deletion failed",
			},
		})
	}
}
