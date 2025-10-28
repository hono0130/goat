package goat

// Rule represents a property that can be enforced during model checking.
// Rules may register conditions, invariants, or temporal specifications.
type Rule interface {
        apply(*options)
}

type ruleFunc func(*options)

func (f ruleFunc) apply(o *options) {
        f(o)
}

// WithRules registers rules for model checking. Rules can include invariants
// (via Always) and temporal properties (via functions such as
// WheneverPEventuallyQ).
func WithRules(rs ...Rule) Option {
        return optionFunc(func(o *options) {
                for _, r := range rs {
                        if r == nil {
                                continue
                        }
                        r.apply(o)
                }
        })
}

// Always registers c as an invariant that must hold in every explored world.
func Always(c Condition) Rule {
        return ruleFunc(func(o *options) {
                registerCondition(o, c)
                o.invariants = append(o.invariants, c.Name())
        })
}

func registerCondition(o *options, c Condition) {
        if c == nil {
                return
        }
        if o.conds == nil {
                o.conds = make(map[ConditionName]Condition)
        }
        o.conds[c.Name()] = c
}

func registerTemporalRule(o *options, rule ltlRule) {
        o.ltlRules = append(o.ltlRules, rule)
}
