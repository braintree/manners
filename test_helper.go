package manners

import (
	"bytes"
	"net/http"
)

func NewResponseWriter() *MyResponseWriter {
	return &MyResponseWriter{MyHeader: make(map[string][]string), Content: bytes.NewBuffer([]byte("content"))}
}

type MyResponseWriter struct {
	MyHeader http.Header
	Content  *bytes.Buffer
}

func (this *MyResponseWriter) Header() http.Header {
	return this.MyHeader
}

func (this *MyResponseWriter) Write(content []byte) (int, error) {
	return this.Content.Write(content)
}

func (this *MyResponseWriter) WriteHeader(status int) {}
