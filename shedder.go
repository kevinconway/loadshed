// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package loadshed

import (
	"context"
	"fmt"
	"math/rand"
)

// ErrRejection provides the details on why an invocation was rejected.
type ErrRejection struct {
	// Rule matches the name of any deterministic rule or the value of
	// RuleProbabilistic if a rejection rate was used.
	Rule string
	// Classification optionally contains the invocations classification value
	// if one is set. It is otherwise empty.
	Classification Classification
	// Name is present when Rule matches RuleProbabilistic and contains the
	// name of the RejectionRate/FailureProbability/Capacity used to make the
	// decision
	Name string
	// Usage is present when Rule matches RuleProbabilistic and contains the
	// current capacity utilization.
	Usage float32
	// Likelihood is present when Rule matches RuleProbabilistic and contains
	// the current likelihood of failure due to the capacity utilization.
	Likelihood float32
	// Rate is present when Rule matches RuleProbabilistic and contains the
	// current rejection rate as derived from the probability of failure.
	Rate float32
}

func (self ErrRejection) Error() string {
	if self.Rule != RuleProbabilistic {
		return fmt.Sprintf("Rejected: Rule(%s) Class(%s)", self.Rule, self.Classification)
	}
	return fmt.Sprintf("Rejected: Rule(%s) Class(%s) %s(Usage(%3.1f%%),Likelihood(%3.1f%%),Rate(%3.1f%%))", self.Rule, self.Classification, self.Name, self.Usage*100, self.Likelihood*100, self.Rate*100)
}

type OptionShedder func(*Shedder)

func OptionShedderRejectionRate(r RejectionRate) OptionShedder {
	return func(s *Shedder) {
		s.rejectionRates = append(s.rejectionRates, r)
	}
}

func OptionShedderRule(r Rule) OptionShedder {
	return func(s *Shedder) {
		s.rules = append(s.rules, r)
	}
}

func OptionShedderClassifier(c Classifier) OptionShedder {
	return func(s *Shedder) {
		s.classifier = c
	}
}

func OptionShedderRandom(r func() float32) OptionShedder {
	return func(s *Shedder) {
		s.randFloat = r
	}
}

// Shedder encapsulates a series of load shedding policies and applies them to
// method invocations.
//
// The primary usage of the Shedder is intended to be the Do method which
// applies all load shedding rules and rejection rates.
type Shedder struct {
	randFloat      func() float32
	rejectionRates []RejectionRate
	rules          []Rule
	classifier     Classifier
}

func NewShedder(options ...OptionShedder) *Shedder {
	s := &Shedder{
		randFloat: rand.Float32,
	}
	for _, opt := range options {
		opt(s)
	}
	return s
}

// Do optionally runs the function based on the current state of the load
// shedding policy configured for the Shedder. In the event that the function
// is not executed the Shedder will return an ErrRejection.
//
// All invocation monitoring wrappers are applied for the Fn is executed.
// If a Classifier is provided then the current classification is added to the
// context before any other action.
func (self *Shedder) Do(ctx context.Context, fn Fn) error {
	if self.classifier != nil {
		ctx = ClassificationToContext(ctx, self.classifier.Classify(ctx))
	}
	if err := self.Select(ctx); err != nil {
		return err
	}
	for _, rate := range self.rejectionRates {
		if w, ok := rate.(Wrapper); ok {
			fn = w.Wrap(fn)
		}
	}
	return fn(ctx)
}

// Select performs the decision making process for the load shedder and
// optionally returns an error indicating that an action should be rejected. The
// returned error, if not nil, is always an ErrRejected instance. This may be
// used in custom load shedding integrations.
//
// Note that the context given must be the same context that would otherwise be
// given to Do. Also note that Select does not apply any Fn wrapping or
// classification so any invocation monitoring, metrics management, and
// classification must be performed externally.
func (self *Shedder) Select(ctx context.Context) error {
	for _, r := range self.rules {
		if r.Reject(ctx) {
			return ErrRejection{
				Rule:           r.Name(ctx),
				Classification: ClassificationFromContext(ctx),
			}
		}
	}
	for _, r := range self.rejectionRates {
		rate := r.Rate(ctx)
		diceRoll := self.randFloat()
		if diceRoll < rate {
			return ErrRejection{
				Rule:           RuleProbabilistic,
				Classification: ClassificationFromContext(ctx),
				Name:           r.Name(ctx),
				Usage:          r.Usage(ctx),
				Likelihood:     r.Likelihood(ctx),
				Rate:           rate,
			}
		}
	}
	return nil
}

// WrapSelect returns a wrapped version of Fn that both applies any invocation
// monitoring required by rejection rate calculators and applies load shedding
// rules.
//
// This differs from the Do method by returning a re-usable Fn. Most usage of
// the shedder should be through the Do method but this method is provided for
// specialized cases where the input parameters for the Fn do not change with
// each invocation. This allows Fn to be called repeatedly without needing to be
// re-wrapped on each invocation.
func (self *Shedder) WrapSelect(fn Fn) Fn {
	for _, rate := range self.rejectionRates {
		if w, ok := rate.(Wrapper); ok {
			fn = w.Wrap(fn)
		}
	}
	return func(ctx context.Context) error {
		if self.classifier != nil {
			ctx = ClassificationToContext(ctx, self.classifier.Classify(ctx))
		}
		if err := self.Select(ctx); err != nil {
			return err
		}
		return fn(ctx)
	}
}

// RuleProbabilistic is the name of the rejection rule that covers all cases of
// using a rejection rate rather than a deterministic rule.
const RuleProbabilistic string = "PROBABILISTIC"
