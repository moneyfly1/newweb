package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageData struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: data})
}

func SuccessMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: message})
}

func SuccessPage(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code: 0, Message: "success",
		Data: PageData{Items: items, Total: total, Page: page, PageSize: pageSize},
	})
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, ErrorResponse{Code: code, Message: message})
}

func ErrorWithFields(c *gin.Context, httpStatus int, code int, message string, errors []FieldError) {
	c.JSON(httpStatus, ErrorResponse{Code: code, Message: message, Errors: errors})
}

func BadRequest(c *gin.Context, message string)     { Error(c, 400, 40000, message) }
func Unauthorized(c *gin.Context, message string)    { Error(c, 401, 40100, message) }
func Forbidden(c *gin.Context, message string)       { Error(c, 403, 40300, message) }
func NotFound(c *gin.Context, message string)        { Error(c, 404, 40400, message) }
func Conflict(c *gin.Context, message string)        { Error(c, 409, 40900, message) }
func TooManyRequests(c *gin.Context, message string) { Error(c, 429, 42900, message) }
func InternalError(c *gin.Context, message string)   { Error(c, 500, 50000, message) }
