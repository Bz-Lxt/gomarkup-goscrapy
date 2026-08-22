package taskflow

import "testing"

func TestTransitions(t *testing.T) {
	if !CanTransit("created", "running") {
		t.Fatal("created -> running")
	}
	if !CanTransit("paused", "running") {
		t.Fatal("paused -> running")
	}
	if CanTransit("succeeded", "running") {
		t.Fatal("succeeded should be terminal")
	}
	if err := Transit("running", "paused"); err != nil {
		t.Fatal(err)
	}
	if err := Transit("cancelled", "running"); err == nil {
		t.Fatal("expected error")
	}
	if !Terminal("failed") || Active("created") {
		t.Fatal("helpers")
	}
}
