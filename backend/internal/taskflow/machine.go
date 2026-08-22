package taskflow

import (
	"fmt"

	"goscrapy/internal/model"
)

var allowed = map[string][]string{
	model.TaskCreated:   {model.TaskRunning, model.TaskCancelled},
	model.TaskRunning:   {model.TaskPaused, model.TaskSucceeded, model.TaskFailed, model.TaskCancelled},
	model.TaskPaused:    {model.TaskRunning, model.TaskCancelled},
	model.TaskSucceeded: {},
	model.TaskFailed:    {},
	model.TaskCancelled: {},
}

var sourcesByDestination = func() map[string][]string {
	out := make(map[string][]string, len(allowed))
	for from, dests := range allowed {
		for _, to := range dests {
			out[to] = append(out[to], from)
		}
	}
	return out
}()

func CanTransit(from, to string) bool {
	for _, next := range allowed[from] {
		if next == to {
			return true
		}
	}
	return false
}

func Transit(from, to string) error {
	if !CanTransit(from, to) {
		return fmt.Errorf("invalid status transition %s -> %s", from, to)
	}
	return nil
}

func Terminal(status string) bool {
	return status == model.TaskSucceeded || status == model.TaskFailed || status == model.TaskCancelled
}

func Active(status string) bool {
	return status == model.TaskRunning
}

func SourcesFor(to string) []string {
	return sourcesByDestination[to]
}
