package state

import (
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ActiveSession != "" || s.ActiveIteration != "" {
		t.Errorf("expected zero state, got %+v", s)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	want := &State{ActiveSession: "q4-review", ActiveIteration: "rehearsal-3"}
	if err := Save(dir, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if *got != *want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSetActiveSession_ClearsIterationOnSwitch(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, &State{ActiveSession: "a", ActiveIteration: "iter-1"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := SetActiveSession(dir, "b"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.ActiveSession != "b" {
		t.Errorf("session = %q, want b", got.ActiveSession)
	}
	if got.ActiveIteration != "" {
		t.Errorf("expected iteration cleared, got %q", got.ActiveIteration)
	}
}

func TestSetActiveSession_PreservesIterationWhenSame(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, &State{ActiveSession: "a", ActiveIteration: "iter-1"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := SetActiveSession(dir, "a"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.ActiveIteration != "iter-1" {
		t.Errorf("expected iteration preserved, got %q", got.ActiveIteration)
	}
}

func TestSetActiveIteration(t *testing.T) {
	dir := t.TempDir()
	if err := SetActiveIteration(dir, "iter-2"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.ActiveIteration != "iter-2" {
		t.Errorf("got %q, want iter-2", got.ActiveIteration)
	}
}
