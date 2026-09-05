package model

import "testing"

func TestDefaultModelStack(t *testing.T) {
	if DefaultPersonaModel != "Qwen/Qwen3-4B-Instruct-2507" {
		t.Fatalf("unexpected persona model: %q", DefaultPersonaModel)
	}
	if DefaultEvaluatorModel != "Qwen/Qwen3-30B-A3B-Instruct-2507" {
		t.Fatalf("unexpected evaluator model: %q", DefaultEvaluatorModel)
	}
}
