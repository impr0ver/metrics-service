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

func NewResponceWriterWithLogging(w http.ResponseWriter) *LoggingResponseWriter {
	responseData := &ResponseData{
		Status: 0,
		Size:   0,
	}

	lw := LoggingResponseWriter{
		ResponseWriter: w,
		ResponseData:   responseData,
	}
	return &lw
}

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.ResponseData.Size += size
	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.ResponseData.Status = statusCode
}
