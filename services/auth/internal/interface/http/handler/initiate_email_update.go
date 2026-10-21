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
		response.RespondJSON(w, http.StatusMethodNotAllowed, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	userIDVal, ok := r.Context().Value("userID").(string)
	if !ok {
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "UNAUTHORIZED_ACTION",
				Message: "user id is undefined",
			},
		})
		return
	}
	userID := strings.TrimSpace(userIDVal)
	if userID == "" {
		response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
			Success: false,
			Errors: &response.APIError{
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

	input := usecase.EmailUpdateInput{
		ID:       userID,
		Password: body.Password,
		NewEmail: strings.TrimSpace(body.NewEmail),
	}

	if err := h.initiateEmailUpdate.Execute(r.Context(), input); err != nil {
		h.handleUsecaseError(w, err)
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
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
		response.RespondJSON(w, http.StatusNotFound, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "USER_NOT_FOUND",
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
	case errors.Is(err, usecase.ErrSameEmail):
		response.RespondJSON(w, http.StatusConflict, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "SAME_EMAIL",
				Message: err.Error(),
			},
		})
	case errors.Is(err, usecase.ErrEmailAlreadyTaken):
		response.RespondJSON(w, http.StatusConflict, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "EMAIL_ALREADY_TAKEN",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrInvalidEmail):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"new_email": err.Error(),
				},
			},
		})
	case errors.Is(err, usecase.ErrVerificationEmailSendFail):
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "EMAIL_DELIVERY_FAILED",
				Message: "failed to send verification email, please try again",
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "email update initiation failed",
			},
		})
	}
}
