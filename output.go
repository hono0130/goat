package goat

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type kripkeSummary struct {
	TotalWorlds int `json:"total_worlds"`

	InvariantViolations struct {
		Found bool `json:"found"`
		Count int  `json:"count"`
	} `json:"invariant_violations"`

	ExecutionTimeMs int64 `json:"execution_time_ms"`
}

func (k *kripke) writeDot(w io.Writer) {
	_, _ = fmt.Fprintln(w, "digraph {")

	// Sort world IDs for deterministic output
	var worldIDs []worldID
	for id := range k.worlds {
		worldIDs = append(worldIDs, id)
	}
	sort.Slice(worldIDs, func(i, j int) bool { return worldIDs[i] < worldIDs[j] })

	for _, id := range worldIDs {
		wld := k.worlds[id]
		_, _ = fmt.Fprintf(w, "  %d [ label=\"%s\" ];\n", id, wld.label())
		if id == k.initial.id {
			_, _ = fmt.Fprintf(w, "  %d [ penwidth=5 ];\n", id)
		}
		if wld.invariantViolation {
			_, _ = fmt.Fprintf(w, "  %d [ color=red, penwidth=3 ];\n", id)
		}
	}

	// Sort accessible edges for deterministic output
	var fromIDs []worldID
	for from := range k.accessible {
		fromIDs = append(fromIDs, from)
	}
	sort.Slice(fromIDs, func(i, j int) bool { return fromIDs[i] < fromIDs[j] })

	for _, from := range fromIDs {
		tos := k.accessible[from]
		sort.Slice(tos, func(i, j int) bool { return tos[i] < tos[j] })
		for _, to := range tos {
			_, _ = fmt.Fprintf(w, "  %d -> %d;\n", from, to)
		}
	}
	_, _ = fmt.Fprintln(w, "}")
}

func (k *kripke) writeLog(w io.Writer, invariantDescription string) {
	paths := k.findPathsToViolations()

	if len(paths) == 0 {
		_, _ = fmt.Fprintln(w, "No invariant violations found.")
		return
	}

	for i, path := range paths {
		if i > 0 {
			_, _ = fmt.Fprintln(w, "")
		}

		_, _ = fmt.Fprintf(w, "InvariantError:  %s   ✘\n", invariantDescription)
		_, _ = fmt.Fprintf(w, "Path (length = %d):\n", len(path))

		for j, worldID := range path {
			world := k.worlds[worldID]

			if j == len(path)-1 && world.invariantViolation {
				_, _ = fmt.Fprintf(w, "  [%d] <-- violation here\n", j)
			} else {
				_, _ = fmt.Fprintf(w, "  [%d]\n", j)
			}
			_, _ = fmt.Fprintf(w, "  StateMachines:\n")
			for _, sm := range world.env.machines {
				_, _ = fmt.Fprintf(w, "    Name: %s, Detail: %s, State: %s\n", getStateMachineName(sm), getStateMachineDetails(sm), getStateDetails(sm.currentState()))
			}
			_, _ = fmt.Fprintf(w, "  QueuedEvents:\n")
			for smID, events := range world.env.queue {
				for _, event := range events {
					_, _ = fmt.Fprintf(w, "    StateMachine: %s, Event: %s, Detail: %s\n", getStateMachineName(world.env.machines[smID]), getEventName(event), getEventDetails(event))
				}
			}
		}
	}
}

func (k *kripke) findPathsToViolations() [][]worldID {
	var paths [][]worldID

	visited := make(map[worldID]bool)

	queue := [][]worldID{{k.initial.id}}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		currentID := path[len(path)-1]

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		if k.worlds[currentID].invariantViolation {
			paths = append(paths, path)
			continue
		}

		for _, nextID := range k.accessible[currentID] {
			if !visited[nextID] {
				newPath := make([]worldID, len(path)+1)
				copy(newPath, path)
				newPath[len(path)] = nextID
				queue = append(queue, newPath)
			}
		}
	}

	return paths
}

func (w world) label() string {
	strs := make([]string, 0)
	strs = append(strs, "StateMachines:")
	smIDs := make([]string, 0)
	for _, sm := range w.env.machines {
		smIDs = append(smIDs, sm.id())
	}
	// StateMachine名でソート（UUIDではなく）
	sort.Slice(smIDs, func(i, j int) bool {
		nameI := getStateMachineName(w.env.machines[smIDs[i]])
		nameJ := getStateMachineName(w.env.machines[smIDs[j]])
		if nameI != nameJ {
			return nameI < nameJ
		}
		return smIDs[i] < smIDs[j]
	})
	for _, name := range smIDs {
		sm := w.env.machines[name]
		strs = append(strs, fmt.Sprintf("* %s=%s;%s", getStateMachineName(sm), getStateMachineDetails(sm), getStateDetails(sm.currentState())))
	}

	strs = append(strs, "\nQueuedEvents:")
	smIDs = make([]string, 0)
	for smID := range w.env.queue {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)
	for _, smID := range smIDs {
		for _, e := range w.env.queue[smID] {
			sm := w.env.machines[smID]
			strs = append(strs, fmt.Sprintf("* %s<<%s;%s", getStateMachineName(sm), getEventName(e), getEventDetails(e)))
		}
	}
	return strings.Join(strs, "\n")
}

// JSON output structures for debugging and testing
type worldJSON struct {
	InvariantViolation bool                `json:"invariant_violation"`
	StateMachines      []stateMachineJSON `json:"state_machines"`
	QueuedEvents       []eventJSON        `json:"queued_events"`
}

type stateMachineJSON struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Details string `json:"details"`
}

type eventJSON struct {
	TargetMachine string `json:"target_machine"`
	EventName     string `json:"event_name"`
	Details       string `json:"details"`
}

func (k *kripke) worldsToJSON() []worldJSON {
	allWorlds := make([]worldJSON, 0, len(k.worlds))
	for _, world := range k.worlds {
		worldJSON := k.worldToJSON(world)
		allWorlds = append(allWorlds, worldJSON)
	}

	sort.Slice(allWorlds, func(i, j int) bool {
		return compareWorlds(allWorlds[i], allWorlds[j])
	})

	return allWorlds
}

func compareWorlds(a, b worldJSON) bool {
	// First compare by invariant violation (false < true)
	if a.InvariantViolation != b.InvariantViolation {
		return !a.InvariantViolation && b.InvariantViolation
	}

	// Compare by state machines
	for i := 0; i < len(a.StateMachines) && i < len(b.StateMachines); i++ {
		if a.StateMachines[i].ID != b.StateMachines[i].ID {
			return a.StateMachines[i].ID < b.StateMachines[i].ID
		}
		if a.StateMachines[i].State != b.StateMachines[i].State {
			return a.StateMachines[i].State < b.StateMachines[i].State
		}
		if a.StateMachines[i].Details != b.StateMachines[i].Details {
			return a.StateMachines[i].Details < b.StateMachines[i].Details
		}
	}
	if len(a.StateMachines) != len(b.StateMachines) {
		return len(a.StateMachines) < len(b.StateMachines)
	}

	// Compare by queued events
	for i := 0; i < len(a.QueuedEvents) && i < len(b.QueuedEvents); i++ {
		if a.QueuedEvents[i].TargetMachine != b.QueuedEvents[i].TargetMachine {
			return a.QueuedEvents[i].TargetMachine < b.QueuedEvents[i].TargetMachine
		}
		if a.QueuedEvents[i].EventName != b.QueuedEvents[i].EventName {
			return a.QueuedEvents[i].EventName < b.QueuedEvents[i].EventName
		}
		if a.QueuedEvents[i].Details != b.QueuedEvents[i].Details {
			return a.QueuedEvents[i].Details < b.QueuedEvents[i].Details
		}
	}
	return len(a.QueuedEvents) < len(b.QueuedEvents)
}

func (*kripke) worldToJSON(w world) worldJSON {
	smIDs := make([]string, 0, len(w.env.machines))
	for smID := range w.env.machines {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)

	stateMachines := make([]stateMachineJSON, 0, len(smIDs))
	for _, smID := range smIDs {
		sm := w.env.machines[smID]
		stateMachines = append(stateMachines, stateMachineJSON{
			ID:      smID,
			Name:    getStateMachineName(sm),
			State:   getStateDetails(sm.currentState()),
			Details: getStateMachineDetails(sm),
		})
	}

	// Collect queued events
	queuedEvents := make([]eventJSON, 0)
	for _, smID := range smIDs {
		if events, ok := w.env.queue[smID]; ok {
			for _, event := range events {
				queuedEvents = append(queuedEvents, eventJSON{
					TargetMachine: getStateMachineName(w.env.machines[smID]),
					EventName:     getEventName(event),
					Details:       getEventDetails(event),
				})
			}
		}
	}

	// Sort queued events deterministically by target machine, then event name, then details
	sort.Slice(queuedEvents, func(i, j int) bool {
		if queuedEvents[i].TargetMachine != queuedEvents[j].TargetMachine {
			return queuedEvents[i].TargetMachine < queuedEvents[j].TargetMachine
		}
		if queuedEvents[i].EventName != queuedEvents[j].EventName {
			return queuedEvents[i].EventName < queuedEvents[j].EventName
		}
		return queuedEvents[i].Details < queuedEvents[j].Details
	})

	return worldJSON{
		InvariantViolation: w.invariantViolation,
		StateMachines:      stateMachines,
		QueuedEvents:       queuedEvents,
	}
}

func (k *kripke) summarize(executionTimeMs int64) *kripkeSummary {
	summary := &kripkeSummary{
		TotalWorlds:     len(k.worlds),
		ExecutionTimeMs: executionTimeMs,
	}

	violationCount := 0
	for _, world := range k.worlds {
		if world.invariantViolation {
			violationCount++
		}
	}

	summary.InvariantViolations.Found = violationCount > 0
	summary.InvariantViolations.Count = violationCount

	return summary
}
