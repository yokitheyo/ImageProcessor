package middleware

import (
	"time"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"
)

func LoggerMiddleware() ginext.HandlerFunc {
	return func(c *ginext.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		zlog.Logger.Info().
			Str("method", method).
			Str("path", path).
			Int("status", status).
			Dur("duration", duration).
			Str("client_ip", c.ClientIP()).
			Msg("HTTP request")
	}
}
