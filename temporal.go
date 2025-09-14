package goat

// Lasso represents a counterexample composed of a prefix and a loop.
type Lasso struct {
	Prefix []worldID `json:"prefix"`
	Loop   []worldID `json:"loop"`
}

// TemporalResult contains the outcome of checking a temporal rule.
type TemporalResult struct {
	Rule  string `json:"rule"`
	Holds bool   `json:"holds"`
	Lasso *Lasso `json:"lasso,omitempty"`
}

// TemporalRule defines a temporal property to be checked against the model.
//
// The `evaluate` method is kept unexported so that temporal rules can only be
// constructed via helper functions within this package.
type TemporalRule interface {
	Name() string
	evaluate(*model, *temporalGraph) (bool, *Lasso)
}

type wheneverPEventuallyQ struct {
	p, q Condition
	name string
}

func (r wheneverPEventuallyQ) Name() string { return r.name }

func (r wheneverPEventuallyQ) evaluate(m *model, tg *temporalGraph) (bool, *Lasso) {
	pName := r.p.Name()
	qName := r.q.Name()

	// Precompute SCCs that contain no q and have a cycle.
	qless := map[int]bool{}
	for idx, comp := range tg.sccs {
		if !comp.hasCycle(m.accessible) {
			continue
		}
		allNotQ := true
		for _, w := range comp.nodes {
			if m.labels[w][qName] {
				allNotQ = false
				break
			}
		}
		if allNotQ {
			qless[idx] = true
		}
	}

	for w := range m.worlds {
		if !m.labels[w][pName] {
			continue
		}

		// BFS from w avoiding q
		queue := []worldID{w}
		parents := map[worldID]worldID{w: w}
		for len(queue) > 0 {
			v := queue[0]
			queue = queue[1:]
			idx := tg.sccIndex[v]
			if qless[idx] {
				pathInit := buildPath(w, tg.parents)
				pathFromP := buildPath(v, parents)
				prefix := append(pathInit, pathFromP[1:]...)
				loop := buildLoop(v, tg.sccs[idx].nodes, m.accessible)
				return false, &Lasso{Prefix: prefix, Loop: loop}
			}
			for _, u := range m.accessible[v] {
				if m.labels[u][qName] {
					continue
				}
				if _, seen := parents[u]; !seen {
					parents[u] = v
					queue = append(queue, u)
				}
			}
		}
	}

	return true, nil
}

type eventuallyAlways struct {
	c    Condition
	name string
}

func (r eventuallyAlways) Name() string { return r.name }

func (r eventuallyAlways) evaluate(m *model, tg *temporalGraph) (bool, *Lasso) {
	cName := r.c.Name()
	for _, comp := range tg.sccs {
		allC := true
		for _, w := range comp.nodes {
			if !m.labels[w][cName] {
				allC = false
				break
			}
		}
		if allC {
			return true, nil
		}
	}

	// violation: find SCC with a cycle containing !c
	for _, comp := range tg.sccs {
		if !comp.hasCycle(m.accessible) {
			continue
		}
		for _, w := range comp.nodes {
			if !m.labels[w][cName] {
				prefix := buildPath(w, tg.parents)
				loop := buildLoop(w, comp.nodes, m.accessible)
				return false, &Lasso{Prefix: prefix, Loop: loop}
			}
		}
	}
	return false, nil
}

type alwaysEventually struct {
	c    Condition
	name string
}

func (r alwaysEventually) Name() string { return r.name }

func (r alwaysEventually) evaluate(m *model, tg *temporalGraph) (bool, *Lasso) {
	cName := r.c.Name()
	for _, comp := range tg.sccs {
		if !comp.hasCycle(m.accessible) {
			continue
		}
		allNotC := true
		for _, w := range comp.nodes {
			if m.labels[w][cName] {
				allNotC = false
				break
			}
		}
		if allNotC {
			start := comp.nodes[0]
			prefix := buildPath(start, tg.parents)
			loop := buildLoop(start, comp.nodes, m.accessible)
			return false, &Lasso{Prefix: prefix, Loop: loop}
		}
	}
	return true, nil
}

// Pattern constructors

// WheneverPEventuallyQ returns a rule representing G (p -> F q).
func WheneverPEventuallyQ(p, q Condition) TemporalRule {
	name := "whenever " + string(p.Name()) + " eventually " + string(q.Name())
	return wheneverPEventuallyQ{p: p, q: q, name: name}
}

// EventuallyAlways returns a rule representing F G c.
func EventuallyAlways(c Condition) TemporalRule {
	name := "eventually always " + string(c.Name())
	return eventuallyAlways{c: c, name: name}
}

// AlwaysEventually returns a rule representing G F c.
func AlwaysEventually(c Condition) TemporalRule {
	name := "always eventually " + string(c.Name())
	return alwaysEventually{c: c, name: name}
}

// internal graph utilities

type scc struct {
	nodes []worldID
}

func (s scc) hasCycle(g map[worldID][]worldID) bool {
	if len(s.nodes) > 1 {
		return true
	}
	w := s.nodes[0]
	for _, n := range g[w] {
		if n == w {
			return true
		}
	}
	return false
}

type temporalGraph struct {
	parents  map[worldID]worldID
	sccs     []scc
	sccIndex map[worldID]int
}

func newTemporalGraph(m *model) *temporalGraph {
	parents := bfsParents(m.initial.id, m.accessible)
	sccs, sccIdx := tarjan(m.initial.id, m.accessible)
	return &temporalGraph{parents: parents, sccs: sccs, sccIndex: sccIdx}
}

func bfsParents(start worldID, g map[worldID][]worldID) map[worldID]worldID {
	parents := map[worldID]worldID{start: start}
	queue := []worldID{start}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		for _, u := range g[v] {
			if _, ok := parents[u]; !ok {
				parents[u] = v
				queue = append(queue, u)
			}
		}
	}
	return parents
}

func tarjan(start worldID, g map[worldID][]worldID) ([]scc, map[worldID]int) {
	index := 0
	indices := make(map[worldID]int)
	lowlink := make(map[worldID]int)
	onStack := make(map[worldID]bool)
	stack := make([]worldID, 0)
	sccs := make([]scc, 0)

	var strongConnect func(v worldID)
	strongConnect = func(v worldID) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range g[v] {
			if _, ok := indices[w]; !ok {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}
		if lowlink[v] == indices[v] {
			comp := make([]worldID, 0)
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc{nodes: comp})
		}
	}

	strongConnect(start)

	sccIdx := make(map[worldID]int)
	for i, comp := range sccs {
		for _, w := range comp.nodes {
			sccIdx[w] = i
		}
	}
	return sccs, sccIdx
}

func buildPath(target worldID, parents map[worldID]worldID) []worldID {
	path := []worldID{}
	cur := target
	for {
		path = append([]worldID{cur}, path...)
		parent := parents[cur]
		if parent == cur {
			break
		}
		cur = parent
	}
	return path
}

func buildLoop(start worldID, nodes []worldID, g map[worldID][]worldID) []worldID {
	set := make(map[worldID]bool)
	for _, n := range nodes {
		set[n] = true
	}
	loop := []worldID{start}
	cur := start
	visited := map[worldID]bool{}
	for {
		visited[cur] = true
		advanced := false
		for _, nxt := range g[cur] {
			if !set[nxt] {
				continue
			}
			loop = append(loop, nxt)
			cur = nxt
			advanced = true
			if nxt == start {
				return loop
			}
			if visited[nxt] {
				return loop
			}
			break
		}
		if !advanced {
			return loop
		}
	}
}

// checkTemporalRules evaluates registered temporal rules.
func (m *model) checkTemporalRules() []TemporalResult {
	if len(m.temporalRules) == 0 {
		return nil
	}
	tg := newTemporalGraph(m)
	results := make([]TemporalResult, 0, len(m.temporalRules))
	for _, r := range m.temporalRules {
		holds, lasso := r.evaluate(m, tg)
		results = append(results, TemporalResult{Rule: r.Name(), Holds: holds, Lasso: lasso})
	}
	return results
}
