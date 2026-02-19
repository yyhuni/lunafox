package dto

import (
	"github.com/gin-gonic/gin"
	common "github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type PaginationQuery = common.PaginationQuery
type PaginatedResponse[T any] = common.PaginatedResponse[T]
type ErrorResponse = common.ErrorResponse
type ErrorBody = common.ErrorBody
type ErrorDetail = common.ErrorDetail

var (
	BindJSON           = common.BindJSON
	BindQuery          = common.BindQuery
	BindURI            = common.BindURI
	Success            = common.Success
	OK                 = common.OK
	Created            = common.Created
	NoContent          = common.NoContent
	Error              = common.Error
	ErrorWithDetails   = common.ErrorWithDetails
	BadRequest         = common.BadRequest
	Unauthorized       = common.Unauthorized
	Forbidden          = common.Forbidden
	NotFound           = common.NotFound
	Conflict           = common.Conflict
	InternalError      = common.InternalError
	ValidationError    = common.ValidationError
	HandleBindingError = common.HandleBindingError
)

func Paginated[T any](c *gin.Context, data []T, total int64, page, pageSize int) {
	common.Paginated(c, data, total, page, pageSize)
}
