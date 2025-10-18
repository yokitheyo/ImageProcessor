package middleware

import (
	"net/http"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/imageprocessor/internal/dto"
)

func ErrorHandlerMiddleware() ginext.HandlerFunc {
	return func(c *ginext.Context) {
		defer func() {
			if err := recover(); err != nil {
				zlog.Logger.Error().
					Interface("error", err).
					Str("path", c.Request.URL.Path).
					Msg("panic recovered")

				c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error:   "internal_error",
					Message: "An internal error occurred",
				})
			}
		}()

		c.Next()
	}
}
