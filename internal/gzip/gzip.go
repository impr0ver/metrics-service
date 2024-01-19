package gzip

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
)

func CompressJSON(w io.Writer, i interface{}) error {
	gz := gzip.NewWriter(w)
	if err := json.NewEncoder(gz).Encode(i); err != nil {
		return err
	}
	return gz.Close()
}

type CompressWriter struct {
	Writer     http.ResponseWriter
	GzipWriter *gzip.Writer
}

func NewCompressWriter(w http.ResponseWriter) *CompressWriter {
	return &CompressWriter{
		Writer:     w,
		GzipWriter: gzip.NewWriter(w),
	}
}

func (c *CompressWriter) Header() http.Header {
	return c.Writer.Header()
}

func (c *CompressWriter) Write(p []byte) (int, error) {
	return c.GzipWriter.Write(p)
}

func (c *CompressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.Writer.Header().Set("Content-Encoding", "gzip")
	}
	c.Writer.WriteHeader(statusCode)
}

func (c *CompressWriter) Close() error {
	return c.GzipWriter.Close()
}

type CompressReader struct {
	Reader     io.ReadCloser
	GzipReader *gzip.Reader
}

func NewCompressReader(r io.ReadCloser) (*CompressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &CompressReader{
		Reader:     r,
		GzipReader: zr,
	}, nil
}

func (c CompressReader) Read(p []byte) (n int, err error) {
	return c.GzipReader.Read(p)
}

func (c *CompressReader) Close() error {
	if err := c.Reader.Close(); err != nil {
		return err
	}
	return c.GzipReader.Close()
}
