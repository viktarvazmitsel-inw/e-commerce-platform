package handler

import (
	"authorization/internal/domain"
	"authorization/internal/usecase"
	"context"
	"errors"
	"net/http"
	"strings"
)

type VerifyUserUseCase interface {
	Execute(ctx context.Context, input usecase.VerificationInput) error
}

type VerifyUserHandler struct {
	verifyUser VerifyUserUseCase
}

type TokenData struct {
	Token string `json:"token"`
}

func NewVerifyUserHandler(uc VerifyUserUseCase) *VerifyUserHandler {
	return &VerifyUserHandler{
		verifyUser: uc,
	}
}

func (h *VerifyUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	var tokenData TokenData
	if DecodeJSON(w, r, &tokenData) {
		return
	}

	token := strings.TrimSpace(tokenData.Token)
	if token == "" {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_VERIFICATION_TOKEN",
				Message: "invalid verification token",
			},
		})
		return
	}

	verificationInput := usecase.VerificationInput{
		Token: token,
	}

	if err := h.verifyUser.Execute(r.Context(), verificationInput); err != nil {
		h.handleUsecaseError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data:    map[string]string{"message": "verification successful"},
	})

}

func (h *VerifyUserHandler) handleUsecaseError(w http.ResponseWriter, err error) {
	errorsDetails := make(map[string]string)

	switch {
	case errors.Is(err, usecase.ErrWrongVerificationToken):
		errorsDetails["token"] = err.Error()
	case errors.Is(err, usecase.ErrUserIdNotFound), errors.Is(err, domain.ErrUserDeactivated):
		errorsDetails["user"] = err.Error()
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "verification failed",
			},
		})
		return
	}

	RespondJSON(w, http.StatusBadRequest, &Response{
		Success: false,
		Errors: &APIError{
			Code:    "VERIFICATION_FAILED",
			Message: "verification failed",
			Details: errorsDetails,
		},
	})
}
