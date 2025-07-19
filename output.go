package goat

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

func (k *kripke) WriteAsDot(w io.Writer) {
	_, _ = fmt.Fprintln(w, "digraph {")
	for id, wld := range k.worlds {
		_, _ = fmt.Fprintf(w, "  %d [ label=\"%s\" ];\n", id, wld.label())
		if id == k.initial.id {
			_, _ = fmt.Fprintf(w, "  %d [ penwidth=5 ];\n", id)
		}
		if wld.invariantViolation {
			_, _ = fmt.Fprintf(w, "  %d [ color=red, penwidth=3 ];\n", id)
		}
	}
	for from, tos := range k.accessible {
		for _, to := range tos {
			_, _ = fmt.Fprintf(w, "  %d -> %d;\n", from, to)
		}
	}
	_, _ = fmt.Fprintln(w, "}")
}

func (k *kripke) WriteAsLog(w io.Writer, invariantDescription string) {
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
type WorldJSON struct {
	InvariantViolation bool               `json:"invariant_violation"`
	StateMachines      []StateMachineJSON `json:"state_machines"`
	QueuedEvents       []EventJSON        `json:"queued_events"`
}

type StateMachineJSON struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Details string `json:"details"`
}

type EventJSON struct {
	TargetMachine string `json:"target_machine"`
	EventName     string `json:"event_name"`
	Details       string `json:"details"`
}

func (k *kripke) WriteWorldsAsJSON(w io.Writer) error {
	allWorlds := make([]WorldJSON, 0, len(k.worlds))
	for _, world := range k.worlds {
		worldJSON := k.worldToJSON(world)
		allWorlds = append(allWorlds, worldJSON)
	}

	sort.Slice(allWorlds, func(i, j int) bool {
		return compareWorlds(allWorlds[i], allWorlds[j])
	})

	result := map[string]interface{}{
		"worlds": allWorlds,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func compareWorlds(a, b WorldJSON) bool {
	var jsonA, jsonB []byte
	jsonA, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	jsonB, err = json.Marshal(b)
	if err != nil {
		panic(err)
	}

	return string(jsonA) < string(jsonB)
}

func (k *kripke) worldToJSON(w world) WorldJSON {
	smIDs := make([]string, 0, len(w.env.machines))
	for smID := range w.env.machines {
		smIDs = append(smIDs, smID)
	}
	sort.Slice(smIDs, func(i, j int) bool {
		detailsI := getStateMachineDetails(w.env.machines[smIDs[i]])
		detailsJ := getStateMachineDetails(w.env.machines[smIDs[j]])
		if detailsI != detailsJ {
			return detailsI < detailsJ
		}
		panic("cannot establish deterministic ordering for state machines")
	})

	stateMachines := make([]StateMachineJSON, 0, len(smIDs))
	for _, smID := range smIDs {
		sm := w.env.machines[smID]
		stateMachines = append(stateMachines, StateMachineJSON{
			Name:    getStateMachineName(sm),
			State:   getStateDetails(sm.currentState()),
			Details: getStateMachineDetails(sm),
		})
	}

	// Collect queued events
	queuedEvents := make([]EventJSON, 0)
	for _, smID := range smIDs {
		if events, ok := w.env.queue[smID]; ok {
			for _, event := range events {
				queuedEvents = append(queuedEvents, EventJSON{
					TargetMachine: getStateMachineName(w.env.machines[smID]),
					EventName:     getEventName(event),
					Details:       getEventDetails(event),
				})
			}
		}
	}

	// Sort queued events deterministically
	sort.Slice(queuedEvents, func(i, j int) bool {
		if queuedEvents[i].Details != queuedEvents[j].Details {
			return queuedEvents[i].Details < queuedEvents[j].Details
		}
		if queuedEvents[i].EventName != queuedEvents[j].EventName {
			return queuedEvents[i].EventName < queuedEvents[j].EventName
		}
		if queuedEvents[i].TargetMachine != queuedEvents[j].TargetMachine {
			return queuedEvents[i].TargetMachine < queuedEvents[j].TargetMachine
		}
		panic("cannot establish deterministic ordering for queued events")
	})

	return WorldJSON{
		InvariantViolation: w.invariantViolation,
		StateMachines:      stateMachines,
		QueuedEvents:       queuedEvents,
	}
}
