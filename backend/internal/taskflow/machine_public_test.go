package taskflow_test

import (
	"testing"

	"goscrapy/internal/model"
	"goscrapy/internal/taskflow"
)

func TestSourcesForReturnsIndependentSnapshot(t *testing.T) {
	first := taskflow.SourcesFor(model.TaskRunning)
	if len(first) == 0 {
		t.Fatal("running state has no source states")
	}
	for i := range first {
		first[i] = model.TaskFailed
	}

	got := taskflow.SourcesFor(model.TaskRunning)
	want := map[string]bool{
		model.TaskCreated: true,
		model.TaskPaused:  true,
	}
	if len(got) != len(want) {
		t.Fatalf("SourcesFor(running) = %v, want created and paused", got)
	}
	for _, state := range got {
		if !want[state] {
			t.Fatalf("SourcesFor(running) = %v after caller modified an earlier result, want created and paused", got)
		}
		delete(want, state)
	}
	if len(want) != 0 {
		t.Fatalf("SourcesFor(running) = %v, missing expected source states: %v", got, want)
	}
}
