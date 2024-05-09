package logger

import (
	"net/http"

	"go.uber.org/zap"
)

type (
	ResponseData struct {
		Status int
		Size   int
	}

	LoggingResponseWriter struct {
		http.ResponseWriter //original http.ResponseWriter
		ResponseData        *ResponseData
	}
)

func NewLogger() *zap.SugaredLogger {

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	//SugaredLogger registrator
	sugar := *logger.Sugar()
	return &sugar
}