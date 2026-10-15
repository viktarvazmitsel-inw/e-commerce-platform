package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
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
		RespondJSON(w, http.StatusMethodNotAllowed, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	userIDVal, ok := r.Context().Value("userID").(string)
	if !ok {
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "author id is undefined",
			},
		})
		return
	}
	authorID := strings.TrimSpace(userIDVal)
	if authorID == "" {
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "author id is undefined",
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
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: "invalid input data",
				Details: errorsDetails,
			},
		})
		return
	}

	softDeleteProfileInput := usecase.SoftDeleteProfileInput{
		AuthorID: authorID,
		TargetID: strings.TrimSpace(inputData.TargetID),
		Password: inputData.Password,
	}

	deletionResult, err := h.deleteProfile.Execute(r.Context(), softDeleteProfileInput)
	if err != nil {
		h.handleUsecaseError(w, err)
		return
	}

	if !deletionResult.SessionRevoked {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "SESSION_HAS_NOT_BEEN_REVOKED",
				Message: "session has not been revoked",
				Details: map[string]string{
					"target_id": inputData.TargetID,
				},
			},
		})
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data: map[string]string{
			"message": "deletion successful",
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
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrUserDeactivated):
		RespondJSON(w, http.StatusForbidden, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "DEACTIVATED_USER",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrInvalidPassword):
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_CREDENTIALS",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrUnauthorizedAction):
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrTargetUserNotFound):
		RespondJSON(w, http.StatusNotFound, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "TARGET_USER_NOT_FOUND",
				Message: err.Error(),
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "deletion failed",
			},
		})
	}
}
