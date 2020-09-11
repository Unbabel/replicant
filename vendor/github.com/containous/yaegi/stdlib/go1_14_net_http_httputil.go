// Code generated by 'github.com/containous/yaegi/extract net/http/httputil'. DO NOT EDIT.

// +build go1.14,!go1.15

package stdlib

import (
	"net/http/httputil"
	"reflect"
)

func init() {
	Symbols["net/http/httputil"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"DumpRequest":               reflect.ValueOf(httputil.DumpRequest),
		"DumpRequestOut":            reflect.ValueOf(httputil.DumpRequestOut),
		"DumpResponse":              reflect.ValueOf(httputil.DumpResponse),
		"ErrClosed":                 reflect.ValueOf(&httputil.ErrClosed).Elem(),
		"ErrLineTooLong":            reflect.ValueOf(&httputil.ErrLineTooLong).Elem(),
		"ErrPersistEOF":             reflect.ValueOf(&httputil.ErrPersistEOF).Elem(),
		"ErrPipeline":               reflect.ValueOf(&httputil.ErrPipeline).Elem(),
		"NewChunkedReader":          reflect.ValueOf(httputil.NewChunkedReader),
		"NewChunkedWriter":          reflect.ValueOf(httputil.NewChunkedWriter),
		"NewClientConn":             reflect.ValueOf(httputil.NewClientConn),
		"NewProxyClientConn":        reflect.ValueOf(httputil.NewProxyClientConn),
		"NewServerConn":             reflect.ValueOf(httputil.NewServerConn),
		"NewSingleHostReverseProxy": reflect.ValueOf(httputil.NewSingleHostReverseProxy),

		// type definitions
		"BufferPool":   reflect.ValueOf((*httputil.BufferPool)(nil)),
		"ClientConn":   reflect.ValueOf((*httputil.ClientConn)(nil)),
		"ReverseProxy": reflect.ValueOf((*httputil.ReverseProxy)(nil)),
		"ServerConn":   reflect.ValueOf((*httputil.ServerConn)(nil)),

		// interface wrapper definitions
		"_BufferPool": reflect.ValueOf((*_net_http_httputil_BufferPool)(nil)),
	}
}

// _net_http_httputil_BufferPool is an interface wrapper for BufferPool type
type _net_http_httputil_BufferPool struct {
	WGet func() []byte
	WPut func(a0 []byte)
}

func (W _net_http_httputil_BufferPool) Get() []byte   { return W.WGet() }
func (W _net_http_httputil_BufferPool) Put(a0 []byte) { W.WPut(a0) }
