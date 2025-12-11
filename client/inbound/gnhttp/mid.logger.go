package gnhttp

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/apix"
	"github.com/jom-io/gorig/global/consts"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
)

var invokeLogger = logger.GetLogger("invoke")

const (
	responseLogKey = "gn_result"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer invokeLogger.Sync()

		htTraceID := c.GetHeader("X-Request-ID")
		if htTraceID != "" {
			c.Set(consts.TraceIDKey, htTraceID)
			newCtx := context.WithValue(c.Request.Context(), consts.TraceIDKey, c.GetString(consts.TraceIDKey))
			c.Request = c.Request.WithContext(newCtx)
		} else {
			apix.SetTraceID(c)
		}

		invokeLogger.Info("IN", doGetArrForIn(c)...)

		c.Next()

		invokeLogger.Info("OUT", doGetArrForOut(c)...)
	}
}

func doGetArrForIn(c *gin.Context) []zap.Field {
	return []zap.Field{
		zap.String(consts.TraceIDKey, apix.GetTraceID(c)),
		zap.String("method", c.Request.Method),
		zap.String("uri", c.Request.RequestURI),
		zap.String("remoteAddr", c.Request.RemoteAddr),
		zap.Any("header", c.Request.Header),
		zap.Any("query", c.Request.URL.Query()),
	}
}

func doGetArrForOut(c *gin.Context) []zap.Field {
	fields := []zap.Field{
		zap.String(consts.TraceIDKey, apix.GetTraceID(c)),
		zap.Int("status", c.Writer.Status()),
	}
	if res, ok := c.Get(responseLogKey); ok {
		fields = append(fields, zap.Any("response", res))
	}
	return fields
}
