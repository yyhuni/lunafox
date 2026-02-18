package httpdto

import (
	"github.com/gin-gonic/gin"
	shared "github.com/yyhuni/lunafox/server/internal/dto"
)

type PaginationQuery = shared.PaginationQuery
type PaginatedResponse[T any] = shared.PaginatedResponse[T]
type ErrorResponse = shared.ErrorResponse
type ErrorBody = shared.ErrorBody
type ErrorDetail = shared.ErrorDetail

var (
	BindJSON           = shared.BindJSON
	BindQuery          = shared.BindQuery
	BindURI            = shared.BindURI
	Success            = shared.Success
	OK                 = shared.OK
	Created            = shared.Created
	NoContent          = shared.NoContent
	Error              = shared.Error
	ErrorWithDetails   = shared.ErrorWithDetails
	BadRequest         = shared.BadRequest
	Unauthorized       = shared.Unauthorized
	Forbidden          = shared.Forbidden
	NotFound           = shared.NotFound
	Conflict           = shared.Conflict
	InternalError      = shared.InternalError
	ValidationError    = shared.ValidationError
	HandleBindingError = shared.HandleBindingError
)

func NewPaginatedResponse[T any](data []T, total int64, page, pageSize int) *PaginatedResponse[T] {
	return shared.NewPaginatedResponse(data, total, page, pageSize)
}

func Paginated[T any](c *gin.Context, data []T, total int64, page, pageSize int) {
	shared.Paginated(c, data, total, page, pageSize)
}
