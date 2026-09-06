package httpdto

import (
	"strings"

	"github.com/gin-gonic/gin"
	shared "github.com/yyhuni/lunafox/server/internal/dto"
)

type PaginationQuery = shared.PaginationQuery
type PaginatedResponse[T any] = shared.PaginatedResponse[T]
type ErrorResponse = shared.ErrorResponse
type ErrorBody = shared.ErrorBody
type ErrorDetail = shared.ErrorDetail

var (
	BindJSON                        = shared.BindJSON
	BindQuery                       = shared.BindQuery
	BindURI                         = shared.BindURI
	Success                         = shared.Success
	OK                              = shared.OK
	Created                         = shared.Created
	NoContent                       = shared.NoContent
	Error                           = shared.Error
	ErrorWithStatus                 = shared.ErrorWithStatus
	ErrorWithContract               = shared.ErrorWithContract
	ErrorWithDetails                = shared.ErrorWithDetails
	ErrorWithTypedDetails           = shared.ErrorWithTypedDetails
	ErrorWithStatusAndTypedDetails  = shared.ErrorWithStatusAndTypedDetails
	WriteWorkflowConfigurationError = shared.WriteWorkflowConfigurationError
	WriteContextError               = shared.WriteContextError
	BadRequest                      = shared.BadRequest
	Unauthorized                    = shared.Unauthorized
	Forbidden                       = shared.Forbidden
	NotFound                        = shared.NotFound
	Conflict                        = shared.Conflict
	InternalError                   = shared.InternalError
	ValidationError                 = shared.ValidationError
	HandleBindingError              = shared.HandleBindingError
)

func NewPaginatedResponse[T any](data []T, total int64, page, pageSize int) *PaginatedResponse[T] {
	return shared.NewPaginatedResponse(data, total, page, pageSize)
}

func Paginated[T any](c *gin.Context, data []T, total int64, page, pageSize int) {
	shared.Paginated(c, data, total, page, pageSize)
}

// DispatchCustomMethod adapts Google-style `:verb` routes to Gin's wildcard
// matcher. Gin cannot register sibling literal colon routes such as
// `/targets:batchCreate` and `/targets:batchDelete`, so callers must expose one
// wildcard route and enumerate the canonical verbs here; do not add legacy
// aliases to the handler map.
func DispatchCustomMethod(c *gin.Context, param string, handlers map[string]gin.HandlerFunc) {
	method := strings.TrimPrefix(c.Param(param), ":")
	if handler, ok := handlers[method]; ok {
		handler(c)
		return
	}
	shared.NotFound(c, "Custom method not found")
}
