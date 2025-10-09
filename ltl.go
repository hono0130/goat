package goat

type baState int

type baTransition struct {
	to   baState
	cond func(map[ConditionName]bool) bool
}

type ba struct {
	initial   baState
	accepting map[baState]bool
	trans     map[baState][]baTransition
}

type lasso struct {
	Prefix []worldID `json:"prefix"`
	Loop   []worldID `json:"loop"`
}

func (*lasso) temporalEvidence() {}

func (m *model) checkLTL() []temporalRuleResult {
	results := make([]temporalRuleResult, 0, len(m.ltlRules))
	for _, r := range m.ltlRules {
		holds, lasso := m.checkBA(r.ba())
		result := temporalRuleResult{Rule: r.name(), Satisfied: holds}
		if lasso != nil {
			result.Evidence = lasso
		}
		results = append(results, result)
	}
	return results
}

type prodNode struct {
	w worldID
	s baState
}

func (m *model) checkBA(b *ba) (bool, *lasso) {
	start := prodNode{w: m.initial.id, s: b.initial}
	graph := make(map[prodNode][]prodNode)
	pre := map[prodNode]prodNode{start: start}
	queue := []prodNode{start}

	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		labels := m.labels[n.w]
		succs := m.accessible[n.w]
		if len(succs) == 0 {
			succs = []worldID{n.w}
		}
		for _, w2 := range succs {
			for _, tr := range b.trans[n.s] {
				if tr.cond(labels) {
					next := prodNode{w: w2, s: tr.to}
					graph[n] = append(graph[n], next)
					if _, ok := pre[next]; !ok {
						pre[next] = n
						queue = append(queue, next)
					}
				}
			}
		}
	}

	for node := range pre {
		if _, ok := graph[node]; !ok {
			graph[node] = nil
		}
	}

	sccs := sccProduct(graph)
	for _, scc := range sccs {
		if !isProdCyclic(scc, graph) {
			continue
		}
		for _, n := range scc {
			if !b.accepting[n.s] {
				continue
			}
			prefix := buildPrefix(pre, n)
			sccSet := make(map[prodNode]bool)
			for _, pn := range scc {
				sccSet[pn] = true
			}
			loop := findCycle(graph, n, sccSet)
			return false, &lasso{Prefix: prefix, Loop: loop}
		}
	}
	return true, nil
}

func buildPrefix(pre map[prodNode]prodNode, to prodNode) []worldID {
	path := []prodNode{to}
	for pre[to] != to {
		to = pre[to]
		path = append([]prodNode{to}, path...)
	}
	res := make([]worldID, len(path))
	for i, n := range path {
		res[i] = n.w
	}
	return res
}

func findCycle(graph map[prodNode][]prodNode, start prodNode, scc map[prodNode]bool) []worldID {
	queue := []prodNode{start}
	pre := map[prodNode]prodNode{start: start}

	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]

		for _, n := range graph[v] {
			if !scc[n] {
				continue
			}

			if n == start {
				path := make([]prodNode, 0)
				x := v
				for x != start {
					path = append([]prodNode{x}, path...)
					x = pre[x]
				}
				path = append([]prodNode{start}, path...)
				loop := make([]worldID, 0, len(path))
				for i := range path {
					loop = append(loop, path[i].w)
				}
				return loop
			}

			if _, seen := pre[n]; seen {
				continue
			}
			pre[n] = v
			queue = append(queue, n)
		}
	}
	return nil
}

func sccProduct(graph map[prodNode][]prodNode) [][]prodNode {
	index := 0
	stack := []prodNode{}
	indices := make(map[prodNode]int)
	lowlink := make(map[prodNode]int)
	onStack := make(map[prodNode]bool)
	var result [][]prodNode

	var strongConnect func(v prodNode)
	strongConnect = func(v prodNode) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range graph[v] {
			if _, ok := indices[w]; !ok {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] && indices[w] < lowlink[v] {
				lowlink[v] = indices[w]
			}
		}

		if lowlink[v] == indices[v] {
			var scc []prodNode
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			result = append(result, scc)
		}
	}

	for v := range graph {
		if _, ok := indices[v]; !ok {
			strongConnect(v)
		}
	}
	return result
}

func isProdCyclic(scc []prodNode, graph map[prodNode][]prodNode) bool {
	if len(scc) > 1 {
		return true
	}
	n := scc[0]
	for _, m := range graph[n] {
		if m == n {
			return true
		}
	}
	return false
}
