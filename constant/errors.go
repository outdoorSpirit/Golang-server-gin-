package constant

import "fmt"

type AppError struct {
	ErrorCode string
	Message   string
	Params    map[string]interface{}
}

func (e AppError) Code() string  { return e.ErrorCode }
func (e AppError) Error() string { return e.Message }

type UnauthorizedError struct{ AppError }

func NewUnauthorizedError(code, message string, params map[string]interface{}) *UnauthorizedError {
	return &UnauthorizedError{
		AppError: AppError{
			ErrorCode: code,
			Message:   message,
			Params:    params,
		},
	}
}

type BadRequestError struct{ AppError }

func NewBadRequestError(code, message string, params map[string]interface{}) *BadRequestError {
	return &BadRequestError{
		AppError: AppError{
			ErrorCode: code,
			Message:   message,
			Params:    params,
		},
	}
}

type NotFoundError struct{ AppError }

func NewNotFoundError(code, message string, params map[string]interface{}) *NotFoundError {
	return &NotFoundError{
		AppError: AppError{
			ErrorCode: code,
			Message:   message,
			Params:    params,
		},
	}
}

type ForbiddenError struct{ AppError }

func NewForbiddenError(code, message string, params map[string]interface{}) *ForbiddenError {
	return &ForbiddenError{
		AppError: AppError{
			ErrorCode: code,
			Message:   message,
			Params:    params,
		},
	}
}

type InternalServerError struct{ AppError }

func NewInternalServerError(code, message string, params map[string]interface{}) *InternalServerError {
	return &InternalServerError{
		AppError: AppError{
			ErrorCode: code,
			Message:   message,
			Params:    params,
		},
	}
}

var (
	INVALID_IMAGE_FORMAT = NewBadRequestError("invalid_image_format", "Invalid image format(png, jpeg and gif are allowed)", nil)
)

func INVALID_NUMBER(number string) *BadRequestError {
	return NewBadRequestError("invalid_number", fmt.Sprintf("Invalid number: %v", number),
		map[string]interface{}{"Number": number})
}

func DB_OPERATION_ERROR(e error) *InternalServerError {
	return NewInternalServerError("db_operation_failed", e.Error(), map[string]interface{}{})
}

func INFLUXDB_OPERATION_ERROR(e error) *InternalServerError {
	return NewInternalServerError("influxdb_operation_failed", e.Error(), map[string]interface{}{})
}