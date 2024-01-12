// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kevinconway/loadshed/v2"
)

func TestHandlerNoShedding(t *testing.T) {
	t.Parallel()

	shed := loadshed.NewShedder()
	middleware := NewHandlerMiddleware(shed)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("middleware did not call wrapped handler: %d", w.Code)
	}
}

func TestHandlerWithShedding(t *testing.T) {
	t.Parallel()

	shed := loadshed.NewShedder(
		loadshed.OptionShedderRule(&staticRule{reject: true}),
	)
	middleware := NewHandlerMiddleware(shed)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("middleware did not shed load. status code: %d", w.Code)
	}
}

func TestHandlerCallback(t *testing.T) {
	t.Parallel()

	shed := loadshed.NewShedder(
		loadshed.OptionShedderRule(&staticRule{reject: true}),
	)
	cb := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}
	middleware := NewHandlerMiddleware(shed, HandlerOptionCallback(http.HandlerFunc(cb)))
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusTeapot {
		t.Fatalf("middleware did not execute custom callback. status code: %d", w.Code)
	}

}

func TestHandlerErrorCode(t *testing.T) {
	t.Parallel()

	shed := loadshed.NewShedder()
	errCode := HandlerOptionErrCodes([]int{http.StatusInternalServerError})
	middleware := NewHandlerMiddleware(shed, errCode)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("middleware did not call wrapped handler: %d", w.Code)
	}
}

func BenchmarkHandlerLoadshedder(b *testing.B) {
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	lower := float32(.50)
	upper := float32(.50)
	bucketDuration := time.Millisecond
	buckets := 5
	requiredPoints := 1
	errorCodes := []int{500}
	cap := loadshed.NewCapacityErrorRate(
		loadshed.OptionErrorRateBucketDuration(bucketDuration),
		loadshed.OptionErrorRateWindowBuckets(buckets),
		loadshed.OptionErrorRateMinimumPoints(requiredPoints),
	)
	prob := loadshed.NewFailureProbabilityCurveIdentity(cap)
	rate := loadshed.NewRejectionRateCurveLinear(prob, lower, upper, 1.0)
	shed := loadshed.NewShedder(
		loadshed.OptionShedderRejectionRate(rate),
	)

	h := NewHandlerMiddleware(shed, HandlerOptionErrCodes(errorCodes))(wrapped)
	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))
	resp := httptest.NewRecorder()

	b.ResetTimer()
	for n := 0; n < b.N; n = n + 1 {
		h.ServeHTTP(resp, req)
	}
}

const nameStatic = "static"

type staticRule struct {
	reject bool
}

func (self *staticRule) Name(ctx context.Context) string {
	return nameStatic
}
func (self *staticRule) Reject(ctx context.Context) bool {
	return self.reject
}
