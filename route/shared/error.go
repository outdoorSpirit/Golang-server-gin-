package shared

import (
	"fmt"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/iancoleman/strcase"
	"github.com/labstack/echo/v4"
	"github.com/spiker/spiker-server/constant"
	"github.com/spiker/spiker-server/lib"
)

const (
	// ErrorCode_ValidationError バリデーションエラーが発生した場合。
	ErrorCode_ValidationError string = "validation_error"
	// ErrorCode_DatabaseError データベースエラーが発生した場合。
	ErrorCode_DatabaseError = "database_error"
)

type ErrorResponse struct {
	StatusCode int               `json:"-"`
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
}

func (e *ErrorResponse) Error() string {
	return e.Message
}

func APIErrorHandler(e error, c echo.Context) {
	handleErrorResponse := func(e *ErrorResponse) {
		if e == nil {
			c.NoContent(http.StatusInternalServerError)
			return
		}
		c.JSON(e.StatusCode, e)
	}

	if !c.Response().Committed {
		localizer := c.Get(ContextI18NLangKey).(*lib.Localizer)

		var errorResponse *ErrorResponse

		if he, ok := e.(*echo.HTTPError); ok {
			errorResponse = &ErrorResponse{
				StatusCode: he.Code,
				Code:       strcase.ToSnake(http.StatusText(he.Code)),
				Message:    fmt.Sprintf("%v", he.Message),
			}
		} else if se, ok := e.(*constant.BadRequestError); ok {
			errorResponse = &ErrorResponse{
				StatusCode: http.StatusBadRequest,
				Code:       se.Code(),
				Message:    localizer.LocalizeWithDefault(se.ErrorCode, se.Params, se.Message),
			}
		} else if se, ok := e.(*constant.UnauthorizedError); ok {
			errorResponse = &ErrorResponse{
				StatusCode: http.StatusUnauthorized,
				Code:       se.Code(),
				Message:    localizer.LocalizeWithDefault(se.ErrorCode, se.Params, se.Message),
			}
		} else if se, ok := e.(*constant.NotFoundError); ok {
			errorResponse = &ErrorResponse{
				StatusCode: http.StatusNotFound,
				Code:       se.Code(),
				Message:    localizer.LocalizeWithDefault(se.ErrorCode, se.Params, se.Message),
			}
		} else if se, ok := e.(*constant.ForbiddenError); ok {
			errorResponse = &ErrorResponse{
				StatusCode: http.StatusForbidden,
				Code:       se.Code(),
				Message:    localizer.LocalizeWithDefault(se.ErrorCode, se.Params, se.Message),
			}
		} else if ve, ok := e.(validation.Errors); ok {
			details := map[string]string{}
			for key, value := range ve {
				details[key] = localizer.Localize(value.Error(), nil)
			}
			errorResponse = &ErrorResponse{
				StatusCode: http.StatusBadRequest,
				Code:       ErrorCode_ValidationError,
				Message:    localizer.Localize("validation error", nil),
				Details:    details,
			}
		} else {
			errorResponse = &ErrorResponse{
				StatusCode: http.StatusInternalServerError,
				Code:       strcase.ToSnake(http.StatusText(http.StatusInternalServerError)),
				Message:    e.Error(),
			}
		}

		handleErrorResponse(errorResponse)
	}
}
