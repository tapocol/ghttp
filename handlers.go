package ghttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

var (
	defaultInvalidJSONPayloadHandler InvalidJSONPayloadHandler = func(err error) (interface{}, int) {
		return "Invalid payload", http.StatusBadGateway
	}
)

type InvalidJSONPayloadHandler func(err error) (interface{}, int)

func SetDefaultInvalidJSONPayloadHandler(fn InvalidJSONPayloadHandler) {
	defaultInvalidJSONPayloadHandler = fn
}

type ResponseTyper interface {
	ResponseType() reflect.Type
}

type PayloadTyper interface {
	PayloadType() reflect.Type
}

type JSONHandlerFunc[O any] func(http.ResponseWriter, *http.Request) (O, int)

type JSONHandler[O any] struct {
	handlerFn JSONHandlerFunc[O]
}

func NewJSONHandler[O any](fn JSONHandlerFunc[O]) JSONHandler[O] {
	return JSONHandler[O]{
		handlerFn: fn,
	}
}

func (h JSONHandler[O]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, statusCode := h.handlerFn(w, r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		fmt.Printf("encoding response body: %+v\n", err)
		return
	}
}

func (h JSONHandler[O]) ResponseType() reflect.Type {
	var v O
	return reflect.TypeOf(v)
}

type JSONPayloadHandlerFunc[I any, O any] func(http.ResponseWriter, *http.Request, I) (O, int)

type JSONPayloadHandler[I any, O any] struct {
	handlerFunc JSONPayloadHandlerFunc[I, O]
}

func NewJSONPayloadHandler[I any, O any](fn JSONPayloadHandlerFunc[I, O]) JSONPayloadHandler[I, O] {
	return JSONPayloadHandler[I, O]{
		handlerFunc: fn,
	}
}

func (h JSONPayloadHandler[I, O]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var resp interface{} // resp will be `O` if using `handlerFunc`
	var statusCode int
	var payload I
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&payload); err == nil {
		resp, statusCode = h.handlerFunc(w, r, payload)
	} else {
		resp, statusCode = defaultInvalidJSONPayloadHandler(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		fmt.Printf("encoding response body: %+v\n", err)
		return
	}
}

func (h JSONPayloadHandlerFunc[I, O]) PayloadType() reflect.Type {
	var v I
	return reflect.TypeOf(v)
}

func (h JSONPayloadHandlerFunc[I, O]) ResponseType() reflect.Type {
	var v O
	return reflect.TypeOf(v)
}
