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

func (h *UpdateUserProfileHandler) HandlerWithUserID(w http.ResponseWriter, r *http.Request, userID string) {
	var body UpdateUserProfileBody
	if !DecodeJSON(w, r, &body) {
		return
	}

	errorDetails := validateUpdateUserProfileBody(&body)
	if len(errorDetails) > 0 {
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: "invalid input data",
				Details: errorDetails,
			},
		})
		return
	}

	if err := h.updateUserProfile.Execute(r.Context(), usecase.UpdateUserProfileInput{
		ID:      userID,
		Name:    body.Name,
		Surname: body.Surname,
		Phone:   body.Phone,
	}); err != nil {
		h.handleUseCaseError(w, err)
		return
	}

	response.RespondJSON(w, http.StatusOK, &response.Response{
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
	case errors.Is(err, usecase.ErrUserEmailIsNotVerified):
		response.RespondJSON(w, http.StatusForbidden, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "UNVERIFIED_EMAIL",
				Message: err.Error(),
			},
		})
	case errors.Is(err, domain.ErrEmptyName):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"name": err.Error(),
				},
			},
		})
	case errors.Is(err, domain.ErrEmptySurname):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"surname": err.Error(),
				},
			},
		})
	case errors.Is(err, domain.ErrInvalidPhone):
		response.RespondJSON(w, http.StatusBadRequest, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INVALID_INPUT_DATA",
				Message: err.Error(),
				Details: map[string]string{
					"phone": err.Error(),
				},
			},
		})
	default:
		response.RespondJSON(w, http.StatusInternalServerError, &response.Response{
			Success: false,
			Errors: &response.APIError{
				Code:    "INTERNAL_ERROR",
				Message: "profile update failed",
			},
		})
	}
}
