# loadshed

**A load shedding tool kit for Go.**

[![Go Reference](https://pkg.go.dev/badge/github.com/kevinconway/loadshed/v2.svg)](https://pkg.go.dev/github.com/kevinconway/loadshed/v2)

- [loadshed](#loadshed)
  - [Overview](#overview)
  - [The Shedder Helper](#the-shedder-helper)
  - [Probabilistic Load Shedding](#probabilistic-load-shedding)
    - [Capacity Usage Metrics](#capacity-usage-metrics)
    - [Failure Probability From Capacity](#failure-probability-from-capacity)
    - [Rejection Rate From Failure Probability](#rejection-rate-from-failure-probability)
  - [Deterministic Load Shedding](#deterministic-load-shedding)
  - [Request Priority Classification](#request-priority-classification)
    - [Using Classification In Rejection Rates](#using-classification-in-rejection-rates)
  - [Standard Library HTTP Integration](#standard-library-http-integration)
    - [HTTP Server](#http-server)
    - [HTTP Client](#http-client)
  - [Installing](#installing)
  - [Development](#development)
  - [Contributors](#contributors)
  - [Fork Of github.com/asecurityteam/loadshed](#fork-of-githubcomasecurityteamloadshed)
    - [Drop-in Replacement Of github.com/asecurityteam/loadshed](#drop-in-replacement-of-githubcomasecurityteamloadshed)
  - [License](#license)

## Overview

Load shedding is the practice of reducing work in a system that is near
overload in order to the protect the system from failure. This project presents
a tool kit or reference implementation of load shedding for Go. I call this a
tool kit because it contains common and re-usable components that can be
configured or arranged in a system rather than a monolithic tool. Load shedding
must always be tailored to a specific system and there are no safe defaults.
This project aims to offer a structured approach to implementing your own load
shedding.

Load shedding is usually determined by a collection of rules or policies rather
than a single factor. The decision making can be separated in to two categories:
deterministic and probabilistic. Deterministic rules may be static or dynamic
in nature and always result in a true or false indicator for shedding load.
Probabilistic policies use estimations or predictions along with an aspect of
chance or randomness to select requests to shed. Both deterministic and
probabilistic rules are often used together to create a functioning system.

This tool kit comes with structured support for deterministic rules,
probabilistic policies, and request categorization that can be used to
prioritize traffic.

## The Shedder Helper

The main interface of the project is a type called `Shedder` that arranges load
shedding rules into a single, easier to use container:

```go
import "github.com/kevinconway/loadshed/v2"

shedder := loadshed.NewShedder()
err := shedder.Do(ctx, myAction)
var rejectionInfo ErrRejection
if err != nil && errors.As(err, &rejectionInfo) {
    fmt.Println(rejectionInfo)
}
```

The `Shedder` can be configured with any number of deterministic rules and
probabilistic policies. If the `Shedder` determines that it should shed load
then the action given to `Do()` is not performed and an error is returned that
details the reasons for the rejection.

`Shedder` can be used directly or see the `stdlib/net/http` package for an
example of how it can be more seamlessly integrated into a system as middleware.

## Probabilistic Load Shedding

Probabilistic polices result in load shedding based on a target rate. For
example, a system may be in a state where it needs to reject 50% of requests or
80% of requests to prevent failure. The most challenging aspect of probabilistic
load shedding is determining the rejection rate which must be tailored to each
specific system and operation.

This project takes an opinionated and structured approach to defining rejection
rates. Rejection rates are calculated as a function of the system's likelihood
of failure. The likelihood of failure is calculated as a function of a specific
resource's capacity utilization.

### Capacity Usage Metrics

Inherent in the concept of load shedding is the concept of capacity, or the
finite availability of a resource that is consumed when doing work. A capacity
in this tool kit is modeled as:
```go
type Capacity interface {
	Name(ctx context.Context) string
	Usage(ctx context.Context) float32
}
```

All metrics used in probabilistic load shedding policies must be mapped to some
upper limit so that their usage can be reported as a value between 0 and 1,
representing the percent utilization. Some metrics have a natural capacity, such
as CPU or memory, that can be used directly. Others such as max concurrency or
queue depth involve an arbitrarily defined limit based on the system's
constraints. For example, a queue depth capacity might look like:

```go
type CapacityQueueDepth[T any] struct {
    Queue []T
    Limit int
}
func(self *CapacityQueueDepth) Name(ctx context.Context) string {
    return "QUEUE DEPTH"
}
func(self *CapacityQueueDepth) Usage(ctx context.Context) float32 {
    return float32(len(self.Queue)) / float32(self.Limit)
}
```

The limit value would be set based on either the design of the system or a value
discovered through testing.

This project contains some pre-built capacity implementations for max
concurrency, error rate, landing rate, and latency or execution time.

### Failure Probability From Capacity

Once a metric is reported as a percent utilization then the next step is to
calculate the likelihood that the system or next request will fail due to
resource exhaustion. Failure probabilities are modeled as:

```go
type FailureProbability interface {
	Capacity
	Likelihood(ctx context.Context) float32
}
```

The likelihood calculation is highly dependent on the specific details of a
system and should be determined through extensive testing. For example, if a
test demonstrates that a system has an increasing risk of catastrophic failure
once the CPU usage exceeds 80% and that risk increases with each point of
utilization beyond 80% then a potential calculation could be:
```go
type CPUFailureProbability struct {
    *CPUCapacity
}
func (self *CPUFailureProbability) Likelihood(ctx context.Context) float32 {
    usage := self.CPUCapacity.Usage(ctx)
    if usage < .8 {
        // No chance of failure due to CPU when under 80% utilization
        return 0.0
    }
    return (usage - .8) / .2 // Linear interpolation between .8 and 1.0
}
```

The example calculates a zero percent chance of failure for any usage value
below 80% and then a linearly increasing risk proportional to the proximity of
the usage to 100% when over 80%. Note that this type of calculation does not
fit for all systems. For systems where a linear progression is desirable then
this formula can be applied to capacities using:
```go
loadshed.NewFailureProbabilityCurveLinear(cap Capacity, lowerThreshold float32, upperThreshold float32, exponent float32)
```

For any calculation, the goal is to convert a percent utilization into a percent
chance of failure.

### Rejection Rate From Failure Probability

The final piece of probabilistic policies is to calculate a rejection rate based
on the likelihood of failure. Rejection rates are modeled as:
```go
type RejectionRate interface {
	FailureProbability
	Rate(ctx context.Context) float32
}
```

A rate calculation requires detailed knowledge of the system and there is no
safe default formula. In systems where resources are consumed equally by all
requests then it may be possible to use the failure probability, directly, as
the rejection rate. This use case is covered by the `NewRejectionRateCurveIdentity`
method that wraps any `FailureProbability` in a `RejectionRate` that simply
returns the `Likelihood()` value.

## Deterministic Load Shedding

Deterministic rules shed traffic based on binary decision making. This decision
making is not necessarily simple or static and may include any amount of
complexity, reference to external state, or even dynamically adjusted values
within a system. The primary difference between deterministic rules and
probabilistic policies is that deterministic rules come to a true/false
conclusion rather than a rate or chance value. Deterministic rules are modeled
as:

```go
type Rule interface {
	Name(ctx context.Context) string
	Reject(ctx context.Context) bool
}
```

The `Shedder` applies all deterministic rules before applying any rejection
rates. There are no pre-built or included rules in the project. Examples of
deterministic policies to integrate include rate limiting, advanced queue
management, and quota enforcement.

## Request Priority Classification

A common practice for load shedding is consider the priority, or classification,
of a request and apply different rules based on that priority. The `Shedder`
can be given an optional classifier in the form of:
```go
type Classification string
type Classifier interface {
	Classify(ctx context.Context) Classification
}
```

A classification is a string identifier of the class or priority of the request.
This project does not include a standard set of classifications so that they
can be tailored to different use cases.

Note that the classifier is only given the request context and not a specific
request type, such as `*http.Request`, so that the tool kit is more broadly
applicable to different kinds of systems. As a consequence of this choice, any
identifying information of a request that is required to determine the
classification must be set in the context. For example, here is how priority
might be set based on a target HTTP URL:
```go
func URLExtractingMiddleware(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        ctx = context.WithValue(ctxKey, r.URL.Path)
        h.ServeHTTP(w, r.WithContext(ctx))
    })
}
const (
    PriorityHigh loadshed.Classification = "HIGH"
    PriorityLow loadshed.Classification = "LOW"
)
type URLClassifier struct {}
func (*URLClassifier) Classify(ctx context.Context) Classification {
    path := ctx.Value(ctxKey)
    if path == nil {
        return PriorityLow
    }
    if strings.HasPrefix(path.(string), "/v2/important/endpoint") {
        return PriorityHigh
    }
    return PriorityLow
}

shedder := loadshed.NewShedder(
    loadshed.OptionShedderClassifier(&URLClassifier{}),
)
var FinalHandler = URLExtractingMiddleware(
    loadshedhttp.NewMiddleware(shedder)( // from the stdlib/net/http sub-package
        originalHandler,
    ),
)
```

### Using Classification In Rejection Rates

The primary purpose of classifying requests is to create a priority hierarchy of
requests that are shed at different rates. For example, a system may have `LOW`,
`NORMAL`, `HIGH`, and `CRITICAL` classifications. As the system approaches a
likelihood of failure then it will start by rejecting only `LOW` priority
requests. If the failure probability continues to increase then it will begin
to reject `NORMAL` requests, and so on.

To support this use case the project includes a
`NewRejectionRateCurveByClassification` constructor that converts a failure
probability into a rejection rate using a unique converting function for each
classification.

## Standard Library HTTP Integration

As both an example integration and a helper for a common case, this project
includes pre-build `Shedder` integrations for the standard library HTTP handler
and client interfaces.

### HTTP Server

The server integration operates as a middleware for `http.Handler` types and
should work with any HTTP mux or framework that targets the standard library
types.
```go
import (
    "github.com/kevinconway/loadshed/v2"
    loadshedhttp "github.com/kevinconway/loadshed/v2/stdlib/net/http"
)

shedder := loadshed.NewShedder()
handler = loadshedhttp.NewMiddleware(shedder)(
    handler,
)
```

The default behavior is to respond with a `503 Service Unavailable` response
code when a request is rejected due to load shedding. This can be modified using
`HandlerOptionCallback` to set a callback that is executed when a request is
rejected. The callback takes the form of an `http.Handler` and is allowed to
perform any logic and respond with any code. Callbacks can also use
`FromHandlerContext` to get the rejection details to, for example, log the
reason why the request was rejected.

If one of the capacities used for load shedding is an error rate then
`HandlerOptionErrCodes` must be used when constructing the middleware. This
option defines which HTTP status codes are considered errors from the
perspective of an error rate. For example, a set of error codes might include
`5xx` but exclude `4xx`.

### HTTP Client

The client integration works very similarly to the server integration except
that it targets the `http.RoundTripper` interface used by the `http.Client`.
```go
import (
    "github.com/kevinconway/loadshed/v2"
    loadshedhttp "github.com/kevinconway/loadshed/v2/stdlib/net/http"
)

shedder := loadshed.NewShedder()
transport := http.DefaultTransport.Clone()
transport = loadshedhttp.NewTransportMiddleware(shedder)(transport)
client := &http.Client{
    Transport: transport,
}
```

When a request is rejected due to load shedding then the client calls will
return an instance of `loadshed.ErrReject` which contains the details behind
why it was rejected. Similar to the server, a custom callback can be installed
to handle the rejection error using `TransportOptionCallback`.

If one of the capacities used for load shedding is an error rate then
`TransportOptionErrorCodes` must be used when constructing the middleware. This
option defines which HTTP status codes are considered errors from the
perspective of an error rate.

## Installing

`go get github.com/kevinconway/loadshed/v2`

## Development

This project has no hard dependencies on any build tools other than Go. You
should be able to run `go test` for any changes and see the results.

If you prefer, the project includes a Makefile with the following rules:

- `update` - Update dependencies in the go.mod file.
- `bin` - Download all optional build and test tools.
- `fmt` - Run `goimports` on all Go source files.
- `test` - Run all tests and create a test coverage record.
- `coverage` - Generate a series of coverage reports from test records.

## Contributors

For bugs or performance improvements, I welcome pull requests, issues, or
comments. If you make a pull request then please be sure to add tests and run
`make fmt`.

For new features, please start a discussion first by creating an issue and
explaining the intended change.

This project includes an integration for the standard library HTTP tools. If
there are other, obvious standard library tools that would benefit from load
shedding then I'd accept an addition to the `stdlib` sub-package. However, I
won't maintain integrations with 3rd party tools in this repository.

## Fork Of github.com/asecurityteam/loadshed

I was part of the original team that built this library for a previous employer.
The project was originally published as bitbucket.org/stride/loadshed and was
later transferred to github.com/asecurityteam/loadshed. Since then, the company
and team priorities have changed and github.com/asecurityteam/loadshed has been
archived.

I have new use cases for this library so I'm maintaining this fork. 

### Drop-in Replacement Of github.com/asecurityteam/loadshed

For convenience, I have created a `v1.2.0` tag that matches the last published
release of github.com/asecurityteam/loadshed. The only difference is that I have
updated the module path to `github.com/kevinconway/loadshed` and replaced the
`github.com/asecurityteam/rolling` dependency with
`github.com/kevinconway/rolling` which I have also forked for the same reasons
as stated above.. You should be able to replace
`github.com/asecurityteam/loadshed` with `github.com/kevinconway/loadshed` in
either your source code or your go.mod using a `replace` directive to pull from
here instead.

There is no particular advantage to doing this, today, unless it's part of a
gradual migration to v2 (the version documented in this file). I do not plan on
porting any bug fixes or performance improvements to v1.

## License

This project is licensed under the Apache 2.0 license. See
[LICENSE.txt](LICENSE.txt) or <http://www.apache.org/licenses/LICENSE-2.0> for
the full terms.

This project is forked from <https://github.com/asecurityteam/loadshed>. Though,
the majority of the current content is actually a modified form of a public, but
unmerged, branch of the project from
<https://bitbucket.org/atlassian/loadshed/src/dev-2.0/>. The original project's
copyright attribution and license terms are:
```
Copyright @ 2017 Atlassian Pty Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

All files are marked with SPDX tags that both attribute the original copyright
as well as identify the author(s) of any significant changes to those files.
