package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/usecase"
)

type InitiateEmailUpdateUseCase interface {
	Execute(ctx context.Context, input usecase.EmailUpdateInput) error
}

type InitiateEmailUpdateHandler struct {
	initiateEmailUpdate InitiateEmailUpdateUseCase
}

type EmailUpdateBody struct {
	Password string `json:"password"`
	NewEmail string `json:"new_email"`
}

func NewInitiateEmailUpdateHandler(uc InitiateEmailUpdateUseCase) *InitiateEmailUpdateHandler {
	return &InitiateEmailUpdateHandler{
		initiateEmailUpdate: uc,
	}
}

func (h *InitiateEmailUpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	userIDVal, ok := r.Context().Value("userID").(string)
	if !ok {
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "user id is undefined",
			},
		})
		return
	}
	userID := strings.TrimSpace(userIDVal)
	if userID == "" {
		RespondJSON(w, http.StatusUnauthorized, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "user id is undefined",
			},
		})
		return
	}

	var body EmailUpdateBody
	if !DecodeJSON(w, r, &body) {
		return
	}

	errorsDetails := validateEmailUpdateBody(&body)
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

	input := usecase.EmailUpdateInput{
		ID:       userID,
		Password: body.Password,
		NewEmail: strings.TrimSpace(body.NewEmail),
	}

	if err := h.initiateEmailUpdate.Execute(r.Context(), input); err != nil {
		h.handleUsecaseError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data: map[string]string{
			"message": "verification email sent to the new address",
		},
	})
}

func validateEmailUpdateBody(body *EmailUpdateBody) map[string]string {
	errorsDetails := make(map[string]string)

	if len(body.Password) < 8 || len(body.Password) > 50 {
		errorsDetails["password"] = "password must be between 8 and 50 characters"
	}

	body.NewEmail = strings.TrimSpace(body.NewEmail)
	body.NewEmail = strings.ToLower(body.NewEmail)
	if !domain.IsEmailValid(body.NewEmail) {
		errorsDetails["new_email"] = "email must be a valid email address"
	}

	return errorsDetails
}

func (h *InitiateEmailUpdateHandler) handleUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserIdNotFound):
		RespondJSON(w, http.StatusNotFound, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "USER_NOT_FOUND",
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
	case errors.Is(err, usecase.ErrSameEmail):
		RespondJSON(w, http.StatusConflict, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "SAME_EMAIL",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrEmailAlreadyTaken):
		RespondJSON(w, http.StatusConflict, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "EMAIL_ALREADY_TAKEN",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrInvalidEmail):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"new_email": err.Error(),
				},
			},
		})
	case errors.Is(err, usecase.ErrVerificationEmailSendFail):
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "EMAIL_DELIVERY_FAILED",
				Message: "failed to send verification email, please try again",
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "email update initiation failed",
			},
		})
	}
}
