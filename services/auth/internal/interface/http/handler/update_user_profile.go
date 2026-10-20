package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/usecase"
)

type UpdateUserProfileUseCase interface {
	Execute(ctx context.Context, input usecase.UpdateUserProfileInput) error
}

type UpdateUserProfileHandler struct {
	updateUserProfile UpdateUserProfileUseCase
}

type UpdateUserProfileBody struct {
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Phone   string `json:"phone"`
}

func NewUpdateUserProfileHandler(uc UpdateUserProfileUseCase) *UpdateUserProfileHandler {
	return &UpdateUserProfileHandler{
		updateUserProfile: uc,
	}
}

func (h *UpdateUserProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var body UpdateUserProfileBody
	if !DecodeJSON(w, r, &body) {
		return
	}

	errorDetails := validateUpdateUserProfileBody(&body)
	if len(errorDetails) > 0 {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: "invalid input data",
				Details: errorDetails,
			},
		})
		return
	}

	err := h.updateUserProfile.Execute(r.Context(), usecase.UpdateUserProfileInput{
		ID:      userID,
		Name:    body.Name,
		Surname: body.Surname,
		Phone:   body.Phone,
	})
	if err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data: map[string]string{
			"message": "profile updated successfully",
		},
	})
}

func validateUpdateUserProfileBody(body *UpdateUserProfileBody) map[string]string {
	errorDetails := make(map[string]string)

	body.Name = strings.TrimSpace(body.Name)
	if !nameRegex.MatchString(body.Name) {
		errorDetails["name"] = "name must consist of two or more letters"
	}

	body.Surname = strings.TrimSpace(body.Surname)
	if body.Surname != "" && !nameRegex.MatchString(body.Surname) {
		errorDetails["surname"] = "surname must consist of two or more letters"
	}

	body.Phone = strings.TrimSpace(body.Phone)
	if !phoneRegex.MatchString(body.Phone) {
		errorDetails["phone"] = "incorrect phone number format"
	}

	return errorDetails
}

func (h *UpdateUserProfileHandler) handleUseCaseError(w http.ResponseWriter, err error) {
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
	case errors.Is(err, usecase.ErrUserEmailIsNotVerified):
		RespondJSON(w, http.StatusForbidden, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "UNVERIFIED_EMAIL",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrEmptyName):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"name": err.Error(),
				},
			},
		})
	case errors.Is(err, domain.ErrEmptySurname):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"surname": err.Error(),
				},
			},
		})
	case errors.Is(err, domain.ErrInvalidPhone):
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"phone": err.Error(),
				},
			},
		})
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "profile update failed",
			},
		})
	}
}
