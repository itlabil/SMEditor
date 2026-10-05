package httpx

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AppError is the only error type handlers map to an HTTP response.
type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

func ErrBadRequest(code, message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: code, Message: message}
}

func ErrNotFound(code, message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: code, Message: message}
}

func ErrConflict(code, message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: code, Message: message}
}

func ErrUnprocessable(code, message string) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: code, Message: message}
}

func ErrUnprocessableDetails(code, message string, details any) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: code, Message: message, Details: details}
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Fail maps err to an HTTP response. AppError is rendered as-is; any other
// error is logged and the client only sees a generic internal_error.
func Fail(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		body := gin.H{"code": appErr.Code, "message": appErr.Message}
		if appErr.Details != nil {
			body["details"] = appErr.Details
		}
		c.JSON(appErr.Status, gin.H{"error": body})
		return
	}
	log.Printf("internal error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
		"code":    "internal_error",
		"message": "Terjadi kesalahan yang tidak terduga",
	}})
}
