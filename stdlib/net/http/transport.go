// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/kevinconway/loadshed/v2"
)

type TransportOption func(*TransportMiddleware) *TransportMiddleware

// TransportOptionErrorCodes sets the HTTP status codes that should be
// considered errors. This option must be given if load shedding is configured
// to operate on error rates.
func TransportOptionErrorCodes(errorCodes []int) TransportOption {
	return func(t *TransportMiddleware) *TransportMiddleware {
		nt := &errCodeTransport{
			wrapped:    t.wrapped,
			errorCodes: make(map[int]bool, len(errorCodes)),
		}
		for _, code := range errorCodes {
			nt.errorCodes[code] = true
		}
		t.wrapped = nt
		return t
	}
}

// TransportOptionCallback sets the load shedding callback. This enables custom
// behavior when a request is rejected.
func TransportOptionCallback(cb func(*http.Request) (*http.Response, error)) TransportOption {
	return func(t *TransportMiddleware) *TransportMiddleware {
		t.callback = cb
		return t
	}
}

// TransportMiddleware is an HTTP client wrapper that applies a load shedding
// policy to outgoing requests.
type TransportMiddleware struct {
	wrapped  http.RoundTripper
	callback func(*http.Request) (*http.Response, error)
	load     *loadshed.Shedder
}

func (c *TransportMiddleware) RoundTrip(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var e = c.load.Do(r.Context(), func(ctx context.Context) error {
		var innerResp, innerEr = c.wrapped.RoundTrip(r.WithContext(ctx)) //nolint:bodyclose
		resp = innerResp
		return innerEr
	})

	status := &errStatusCode{}
	if errors.As(e, &status) {
		// Reset error to nil on the return value because this would normally
		// not be an error condition. This error is already recorded by any
		// load shed related wrappers and is no longer needed. Doing this masks
		// the fact that we're juggling the status code error.
		e = nil
	}
	shed := loadshed.ErrRejection{}
	if errors.As(e, &shed) {
		if c.callback != nil {
			r = r.WithContext(NewTransportContext(r.Context(), shed))
			return c.callback(r)
		}
	}
	return resp, e
}

func NewTransportMiddleware(shed *loadshed.Shedder, options ...TransportOption) func(c http.RoundTripper) http.RoundTripper {
	return func(c http.RoundTripper) http.RoundTripper {
		var t = &TransportMiddleware{wrapped: c, load: shed}
		for _, option := range options {
			t = option(t)
		}
		return t
	}
}

type ctxKeyTransportType struct{}

var ctxKeyTransport = &ctxKeyTransportType{} //nolint:gochecknoglobals

// NewTransportContext inserts rejection details into the context after a
// request has been rejected.
func NewTransportContext(ctx context.Context, val loadshed.ErrRejection) context.Context {
	return context.WithValue(ctx, ctxKeyTransport, val)
}

// FromTransportContext extracts rejection details from the context after a
// request has been rejected.
func FromTransportContext(ctx context.Context) loadshed.ErrRejection {
	if v, ok := ctx.Value(ctxKeyTransport).(loadshed.ErrRejection); ok {
		return v
	}
	return loadshed.ErrRejection{}
}

// errCodeTransport is an HTTP client wrapper that returns error if the Response
// contains one of the err codes
type errCodeTransport struct {
	wrapped    http.RoundTripper
	errorCodes map[int]bool
}

func (c *errCodeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var er error

	resp, er = c.wrapped.RoundTrip(r)

	if er != nil {
		return resp, er
	}
	if c.errorCodes[resp.StatusCode] {
		return resp, &errStatusCode{errCode: resp.StatusCode} // return error if matches expected error codes
	}
	return resp, er

}
