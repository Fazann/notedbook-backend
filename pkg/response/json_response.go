package response

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/validation"
	"github.com/gin-gonic/gin"
)

// SuccessRes is the body of every successful response. In Swagger docs, set the data type with
// response.SuccessRes{data=dto.XxxRes}.
type SuccessRes struct {
	Message    string      `json:"message" example:"SUCCESS"`
	Data       interface{} `json:"data"`
	StatusCode int         `json:"status_code" example:"200"`
}

// ErrorRes is the body of every error response.
type ErrorRes struct {
	// Code is the translation key (e.g. RECORD_NOT_FOUND), stable for clients to branch on.
	Code       i18n.Key `json:"code" swaggertype:"string" example:"RECORD_NOT_FOUND"`
	Message    string   `json:"message" example:"The record was not found."`
	StatusCode int      `json:"status_code" example:"404"`
	// Errors lists every invalid field; only set when Code is VALIDATION_FAILED.
	Errors []FieldErrorRes `json:"errors,omitempty"`
}

// FieldErrorRes is one invalid request field.
type FieldErrorRes struct {
	Field   string   `json:"field" example:"device.os"`
	Code    i18n.Key `json:"code" swaggertype:"string" example:"FIELD_ONE_OF"`
	Message string   `json:"message" example:"device.os must be one of: IOS, ANDROID, WEB, WINDOW, MAC, LINUX."`
}

type Paginate struct {
	StatusCode int         `json:"status_code"`
	IsNext     bool        `json:"is_next"`
	IsPrev     bool        `json:"is_prev"`
	Data       interface{} `json:"data"`
	Message    string      `json:"message"`
	Total      int64       `json:"total"`
	TotalPage  int         `json:"total_page"`
}

// JsonErrRes writes an error response translated into the request's language
// (`?lang=` or Accept-Language: en, km, ms).
//
// err should be an i18n key (i18n.RecordNotFound) or i18n.Err(key, params). Any other error is
// logged and answered with the generic message for statusCode, so internal details never leak.
// A binding error (c.ShouldBindJSON etc.) with a 400 status is answered with VALIDATION_FAILED
// and one entry per invalid field; the message is the first field's message.
func JsonErrRes(c *gin.Context, err error, statusCode int) {
	lang := i18n.FromRequest(c.Request)
	c.Header("Content-Language", string(lang))

	if statusCode == http.StatusBadRequest {
		if fields, ok := validation.FromError(err); ok && len(fields) > 0 {
			res := ErrorRes{Code: i18n.ValidationFailed, StatusCode: statusCode}
			for _, f := range fields {
				res.Errors = append(res.Errors, FieldErrorRes{
					Field:   f.Field,
					Code:    f.Key,
					Message: i18n.T(lang, f.Key, f.Params),
				})
			}
			res.Message = res.Errors[0].Message
			res.Errors = nil
			c.JSON(statusCode, res)
			return
		}
	}

	key, params, ok := i18n.FromError(err)
	if !ok {
		key = i18n.KeyForStatus(statusCode)
		if statusCode >= http.StatusInternalServerError {
			logger.Error("internal server error", "err", err, "method", c.Request.Method, "path", c.FullPath())
		} else {
			logger.Warn("request error without translation key", "err", err, "status", statusCode, "path", c.FullPath())
		}
	}

	c.JSON(statusCode, ErrorRes{
		Code:       key,
		Message:    i18n.T(lang, key, params),
		StatusCode: statusCode,
	})
}

func JsonSuccessRes(c *gin.Context, data any, statusCode int) {
	if data == nil {
		data = map[string]interface{}{}
	}
	c.JSON(statusCode, SuccessRes{
		Message:    "SUCCESS",
		Data:       data,
		StatusCode: statusCode,
	})
}

func JsonSuccessResWithPage(c *gin.Context, res SvcResWithPaginate) {
	c.JSON(res.StatusCode, Paginate{
		Message:    "SUCCESS",
		IsNext:     res.IsNext,
		IsPrev:     res.IsPrev,
		Data:       res.Data,
		Total:      res.Total,
		StatusCode: res.StatusCode,
		TotalPage:  res.TotalPage,
	})
}
