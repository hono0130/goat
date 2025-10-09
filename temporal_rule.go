package goat

import "fmt"

// TemporalRule represents a temporal property specified
type TemporalRule interface {
	name() string
	isTemporalRule() bool
}

type ltlRule struct {
	n string
	b *ba
}

func (r ltlRule) name() string       { return r.n }
func (ltlRule) isTemporalRule() bool { return true }
func (r ltlRule) ba() *ba            { return r.b }

type temporalEvidence interface {
	temporalEvidence()
}

type temporalRuleResult struct {
	Rule      string           `json:"rule"`
	Satisfied bool             `json:"satisfied"`
	Evidence  temporalEvidence `json:"evidence,omitempty"`
}

// WithTemporalRules registers temporal rules for model checking.
//
// Parameters:
//   - rs: The temporal rules to enforce during verification
//
// Returns an Option that can be passed to Test or Debug.
//
// Example:
//
//	err := goat.Test(
//	    goat.WithStateMachines(sm),
//	    goat.WithTemporalRules(goat.EventuallyAlways(cond)),
//	)
func WithTemporalRules(rs ...TemporalRule) Option {
	return optionFunc(func(o *options) {
		for _, r := range rs {
			ltlRule, ok := r.(ltlRule)
			if !ok {
				panic(fmt.Sprintf("temporal rule %T must be constructed using helper functions like WheneverPEventuallyQ or EventuallyAlways", r))
			}
			o.ltlRules = append(o.ltlRules, ltlRule)
		}
	})
}

// WheneverPEventuallyQ returns a rule enforcing that whenever p holds, q eventually holds.
//
// Parameters:
//   - p: Condition that triggers the obligation
//   - q: Condition that must eventually become true
//
// Returns a TemporalRule that can be registered with WithTemporalRules.
//
// Example:
//
//	err := goat.Test(
//		goat.WithStateMachines(primary, replica),
//		goat.WithConditions(write, replicated),
//		goat.WithTemporalRules(
//			goat.WheneverPEventuallyQ(write, replicated),
//		),
//	)
func WheneverPEventuallyQ(p, q Condition) TemporalRule {
	name := fmt.Sprintf("whenever %s eventually %s", p.Name(), q.Name())
	b := &ba{
		initial:   0,
		accepting: map[baState]bool{1: true},
		trans: map[baState][]baTransition{
			0: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return l[p.Name()] && !l[q.Name()] }},
				{to: 0, cond: func(l map[ConditionName]bool) bool { return !l[p.Name()] || l[q.Name()] }},
			},
			1: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[q.Name()] }},
				{to: 2, cond: func(l map[ConditionName]bool) bool { return l[q.Name()] }},
			},
			2: {
				{to: 2, cond: func(map[ConditionName]bool) bool { return true }},
			},
		},
	}
	return ltlRule{n: name, b: b}
}

// EventuallyAlways returns a rule enforcing that c eventually holds forever.
//
// Parameters:
//   - c: Condition that must eventually remain true
//
// Returns a TemporalRule that can be registered with WithTemporalRules.
//
// Example:
//
//	err := goat.Test(
//	    goat.WithStateMachines(nodes...),
//	    goat.WithConditions(stable),
//	    goat.WithTemporalRules(
//			goat.EventuallyAlways(stable),
//		),
//	)
func EventuallyAlways(c Condition) TemporalRule {
	name := fmt.Sprintf("eventually always %s", c.Name())
	b := &ba{
		initial:   0,
		accepting: map[baState]bool{1: true},
		trans: map[baState][]baTransition{
			0: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[c.Name()] }},
				{to: 0, cond: func(l map[ConditionName]bool) bool { return l[c.Name()] }},
			},
			1: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[c.Name()] }},
				{to: 0, cond: func(l map[ConditionName]bool) bool { return l[c.Name()] }},
			},
		},
	}
	return ltlRule{n: name, b: b}
}

// AlwaysEventually returns a rule enforcing that c holds infinitely often.
//
// Parameters:
//   - c: Condition that must recur indefinitely
//
// Returns a TemporalRule that can be registered with WithTemporalRules.
//
// Example:
//
//	err := goat.Test(
//	    goat.WithStateMachines(node),
//	    goat.WithConditions(heartbeat),
//	    goat.WithTemporalRules(
//			goat.AlwaysEventually(heartbeat),
//		),
//	)
func AlwaysEventually(c Condition) TemporalRule {
	name := fmt.Sprintf("always eventually %s", c.Name())
	b := &ba{
		initial:   0,
		accepting: map[baState]bool{1: true},
		trans: map[baState][]baTransition{
			0: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[c.Name()] }},
				{to: 0, cond: func(map[ConditionName]bool) bool { return true }},
			},
			1: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[c.Name()] }},
				{to: 2, cond: func(l map[ConditionName]bool) bool { return l[c.Name()] }},
			},
			2: {
				{to: 2, cond: func(map[ConditionName]bool) bool { return true }},
			},
		},
	}
	return ltlRule{n: name, b: b}
}
