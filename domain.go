// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import "context"

// Fn is the basic unit of execution and represents an action that may be shed
// under load.
type Fn func(context.Context) error

// Capacity represents usage of some finite resource.
type Capacity interface {
	// Name of the capacity or metric being tracked.
	Name(ctx context.Context) string
	// Usage as a percent value. This should report value between 0 and 1 but
	// some implementations may intentionally report negative or greater 100%
	// values if needed.
	Usage(ctx context.Context) float32
}

// FailureProbability represents the chance of failure based on capacity usage.
//
// Implementations of FailureProbability that wrap or otherwise do not directly
// implement Capacity must account for the wrapped Capacity's optional
// Wrapper interface.
type FailureProbability interface {
	Capacity
	// Likelihood computes a chance of either system or action failure based on
	// the current capacity usage. Values a percentage and should be bounded
	// between 0 and 1. Greater than 100% probability of failure is not
	// particularly meaningful but may have use in some specific scenarios.
	Likelihood(ctx context.Context) float32
}

// RejectionRate represents the amount of load that should be shed based on the
// current failure probability.
//
// Implementations of RejectionRate that wrap or otherwise do not directly
// implement FailureProbability must account for the wrapped
// FailureProbability's optional Wrapper interface.
type RejectionRate interface {
	FailureProbability
	// Rate compute the percentage of load to shed based on the current failure
	// probability. Outputs are expected to be percentage values between 0 and
	// 1. Values outside of this range may result in unexpected behavior.
	Rate(ctx context.Context) float32
}

// Rule represents a deterministic load shedding decision. Unlike RejectionRate,
// a Rule does not incorporate randomness or probability.
//
// Rules can represent virtually any kind of deterministic behavior. For
// example, rules may be used to integrate rate limiting or quota management
// policies into the load shedding framework. Rules also do not have to be
// static. They reference dynamic variables and consult external systems.
type Rule interface {
	Name(ctx context.Context) string
	Reject(ctx context.Context) bool
}

// Wrapper is an optional interface that any Capcity and Probability may
// implement if they need to collect data from functions being executed. For
// example, if a Capcity needs to record the execution duration of all
// function executed within a load shedding policy then it can implement this
// interface by returning a wrapped copy of the passed in function that tracks
// the start and end times.
//
// Implementing this behaviour is optional and this interface is only exposed
// for documentation purposes.
type Wrapper interface {
	Wrap(Fn) Fn
}

// Curve is a function used to scale or plot a value. The primary use cases for
// a curving function is to either translate a capacity usage to a failure
// probability or translate a failure probability to a rejection rate. The
// general expectation is that most inputs and outputes are between the values
// 0 and 1, though specialized use cases may handle arbitrary values.
type Curve interface {
	Curve(ctx context.Context, value float32) float32
}

// Classification is an arbitrary class or category assigned to method
// invocations. The most common use of this type is to define priorities that
// can then be mapped to specific load shedding policies. For example, you might
// define LOW, NORMAL, HIGH, and CRITICAL as classifications and have LOW shed
// at a higher rate than NORMAL, etc.
type Classification string

// Classifier determines the classification of a method invocation.
type Classifier interface {
	Classify(ctx context.Context) Classification
}
