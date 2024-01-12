// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/kevinconway/loadshed/v2"
)

func TestNoErrorCode(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	wrapped := &fixtureTransport{Response: resp, Err: nil}
	tr := &errCodeTransport{
		wrapped:    wrapped,
		errorCodes: make(map[int]bool),
	}
	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))

	_, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func TestDoesNotMatchErrorCode(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	tr := &errCodeTransport{
		wrapped: wrapped,
		errorCodes: map[int]bool{
			500: true,
			501: true,
			502: true,
			503: true,
		},
	}
	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))

	_, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
}

func TestMatchesErrorCode(t *testing.T) {
	t.Parallel()

	expectCode := 500
	resp := &http.Response{
		Status:     "OK",
		StatusCode: expectCode,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	var wrapped = &fixtureTransport{Response: resp, Err: nil}
	tr := &errCodeTransport{
		wrapped: wrapped,
		errorCodes: map[int]bool{
			500: true,
			501: true,
			502: true,
			503: true,
		},
	}
	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))

	_, err := tr.RoundTrip(req)
	e := &errStatusCode{}
	if !errors.As(err, &e) {
		t.Fatalf("expected code error got %T(%s)", err, err)
	}
	if e.errCode != expectCode {
		t.Fatalf("expected code %d but got %d", expectCode, e.errCode)
	}
}

func TestTransport(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	wrapped := &fixtureTransport{Response: resp, Err: nil}
	load := loadshed.NewShedder()
	tr := NewTransportMiddleware(load)(wrapped)

	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))
	_, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error %s", err.Error())
	}
}

func TestTransportErrorRejected(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}
	load := loadshed.NewShedder(
		loadshed.OptionShedderRule(&staticRule{reject: true}),
	)

	wrapped := &fixtureTransport{Response: resp, Err: nil}
	tr := NewTransportMiddleware(load)(wrapped)

	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))
	_, err := tr.RoundTrip(req)
	e := loadshed.ErrRejection{}
	if !errors.As(err, &e) {
		t.Fatalf("Did not get expected error: %T(%s)", e, e)
	}
}

func TestTransportErrCodeOption(t *testing.T) {
	t.Parallel()

	load := loadshed.NewShedder()
	resp := &http.Response{
		Status:     "OK",
		StatusCode: 500,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	wrapped := &fixtureTransport{Response: resp, Err: nil}
	errOption := TransportOptionErrorCodes([]int{500, 501})
	tr := NewTransportMiddleware(load, errOption)(wrapped)

	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))
	res, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %T(%s)", err, err)
	}
	if res.StatusCode != 500 {
		t.Fatalf("expected 500 status but got %d", resp.StatusCode)
	}
}

func TestTransportOptionCallback(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}
	load := loadshed.NewShedder(
		loadshed.OptionShedderRule(&staticRule{reject: true}),
	)

	counter := 0
	cbErr := errors.New("test")
	cb := func(*http.Request) (*http.Response, error) {
		counter++
		return nil, cbErr
	}

	wrapped := &fixtureTransport{Response: resp, Err: nil}
	cbOption := TransportOptionCallback(cb)
	tr := NewTransportMiddleware(load, cbOption)(wrapped)

	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))
	_, err := tr.RoundTrip(req)
	if err != cbErr {
		t.Fatalf("expected the callback error but got %+v", err)
	}
	if counter != 1 {
		t.Fatalf("expected 1 callback but got %d", counter)
	}
}

func TestTransportLoadshedder(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		Status:     "OK",
		StatusCode: 500,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	wrapped := &fixtureTransport{Response: resp, Err: nil}
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

	tr := NewTransportMiddleware(shed, TransportOptionErrorCodes(errorCodes))(wrapped)

	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))
	res, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error %T(%s)", err, err)
	}
	if res.StatusCode != 500 {
		t.Fatalf("expected a 500 status but got %d", res.StatusCode)
	}

	_, err = tr.RoundTrip(req)
	e := loadshed.ErrRejection{}
	if !errors.As(err, &e) {
		t.Fatalf("expected a load shed error but got %T(%s)", err, err)
	}
}

var benchTransportLoadshedderResult *http.Response
var benchTransportLoadshedderError error

func BenchmarkTransportLoadshedder(b *testing.B) {
	resp := &http.Response{
		Status:     "Internal Server Error",
		StatusCode: 500,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	wrapped := &fixtureTransport{Response: resp, Err: nil}
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

	tr := NewTransportMiddleware(shed, TransportOptionErrorCodes(errorCodes))(wrapped)
	req, _ := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(``))))

	b.ResetTimer()
	for n := 0; n < b.N; n = n + 1 {
		benchTransportLoadshedderResult, benchTransportLoadshedderError = tr.RoundTrip(req)
	}
}

type fixtureTransport struct {
	Response *http.Response
	Err      error
}

func (c *fixtureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return c.Response, c.Err
}
