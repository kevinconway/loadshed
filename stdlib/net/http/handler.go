// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/felixge/httpsnoop"

	"github.com/kevinconway/loadshed/v2"
)

// HandlerOption represents configuration for the http.Handler middleware.
type HandlerOption func(*HandlerMiddleware) *HandlerMiddleware

// HandlerOptionCallback adds a callback to the middleware that is invoked
// each time the load shedder rejects a request. This can be used to collect
// load shedder metrics or to respond with a custom status and message.
func HandlerOptionCallback(cb http.Handler) HandlerOption {
	return func(m *HandlerMiddleware) *HandlerMiddleware {
		m.callback = cb
		return m
	}
}

// HandlerOptionErrCodes determines which status codes result in errors that
// are reported to the load shedder. Setting this is required if you plan to
// shed load based on error rates.
func HandlerOptionErrCodes(errCodes []int) HandlerOption {
	return func(m *HandlerMiddleware) *HandlerMiddleware {
		m.errCodes = make(map[int]bool, len(errCodes))
		for _, code := range errCodes {
			m.errCodes[code] = true
		}
		return m
	}
}

// HandlerMiddleware struct represents an HTTP handler middleware that applies
// a load shedding policy to incoming requests.
type HandlerMiddleware struct {
	next     http.Handler
	errCodes map[int]bool
	shed     *loadshed.Shedder
	callback http.Handler
}

func (m *HandlerMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := 200
	proxy := httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(whf httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(code int) {
				whf(code)
				status = code
			}
		},
	})

	var lerr = m.shed.Do(r.Context(), func(ctx context.Context) error {
		m.next.ServeHTTP(proxy, r)
		if m.errCodes[status] {
			return &errStatusCode{errCode: status}
		}
		return nil
	})

	shed := loadshed.ErrRejection{}
	if errors.As(lerr, &shed) {
		r = r.WithContext(NewHandlerContext(r.Context(), shed))
		m.callback.ServeHTTP(proxy, r)
	}
}

func defaultCallback(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

func NewHandlerMiddleware(l *loadshed.Shedder, options ...HandlerOption) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		var m = &HandlerMiddleware{
			next:     next,
			shed:     l,
			callback: http.HandlerFunc(defaultCallback),
		}
		for _, option := range options {
			m = option(m)
		}
		return m
	}
}

type ctxKeyHandlerType struct{}

var ctxKeyHandler = &ctxKeyHandlerType{} //nolint:gochecknoglobals

// NewHandlerContext inserts rejection details into the context after a request
// has been rejected.
func NewHandlerContext(ctx context.Context, val loadshed.ErrRejection) context.Context {
	return context.WithValue(ctx, ctxKeyHandler, val)
}

// FromHandlerContext extracts rejection details from the context after a
// request has been rejected.
func FromHandlerContext(ctx context.Context) loadshed.ErrRejection {
	if v, ok := ctx.Value(ctxKeyHandler).(loadshed.ErrRejection); ok {
		return v
	}
	return loadshed.ErrRejection{}
}
