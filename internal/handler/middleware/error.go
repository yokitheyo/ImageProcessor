package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/dto"
)

func ErrorHandlerMiddleware() ginext.HandlerFunc {
	return func(c *ginext.Context) {
		defer func() {
			if err := recover(); err != nil {
				zlog.Logger.Error().
					Str("error", fmt.Sprintf("%v", err)).
					Str("path", c.Request.URL.Path).
					Msg("panic recovered")

				zlog.Logger.Error().Msgf("stacktrace:\n%s", string(debug.Stack()))

				c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error:   "internal_error",
					Message: "An internal error occurred",
				})
			}
		}()

		c.Next()
	}
}
