package goat

import "testing"

func TestAlwaysEventuallyViolation(t *testing.T) {
	c := BoolCondition("c", false)
	w0 := world{id: 0}
	w1 := world{id: 1}
	m := model{
		worlds:        worlds{0: w0, 1: w1},
		initial:       w0,
		accessible:    map[worldID][]worldID{0: {1}, 1: {1}},
		conds:         map[ConditionName]Condition{c.Name(): c},
		labels:        map[worldID]map[ConditionName]bool{0: {c.Name(): false}, 1: {c.Name(): false}},
		temporalRules: []TemporalRule{AlwaysEventually(c)},
	}
	res := m.checkTemporalRules()
	if len(res) != 1 || res[0].Holds {
		t.Fatalf("expected violation, got %v", res)
	}
}

func TestEventuallyAlwaysHold(t *testing.T) {
	c := BoolCondition("c", true)
	w0 := world{id: 0}
	w1 := world{id: 1}
	m := model{
		worlds:     worlds{0: w0, 1: w1},
		initial:    w0,
		accessible: map[worldID][]worldID{0: {1}, 1: {1}},
		conds:      map[ConditionName]Condition{c.Name(): c},
		labels: map[worldID]map[ConditionName]bool{
			0: {c.Name(): false},
			1: {c.Name(): true},
		},
		temporalRules: []TemporalRule{EventuallyAlways(c)},
	}
	res := m.checkTemporalRules()
	if len(res) != 1 || !res[0].Holds {
		t.Fatalf("expected rule to hold, got %v", res)
	}
}

func TestWheneverPEventuallyQViolation(t *testing.T) {
	p := BoolCondition("p", true)
	q := BoolCondition("q", false)
	w0 := world{id: 0}
	w1 := world{id: 1}
	m := model{
		worlds:     worlds{0: w0, 1: w1},
		initial:    w0,
		accessible: map[worldID][]worldID{0: {1}, 1: {1}},
		conds: map[ConditionName]Condition{
			p.Name(): p,
			q.Name(): q,
		},
		labels: map[worldID]map[ConditionName]bool{
			0: {p.Name(): true, q.Name(): false},
			1: {p.Name(): false, q.Name(): false},
		},
		temporalRules: []TemporalRule{WheneverPEventuallyQ(p, q)},
	}
	res := m.checkTemporalRules()
	if len(res) != 1 || res[0].Holds {
		t.Fatalf("expected violation, got %v", res)
	}
}
