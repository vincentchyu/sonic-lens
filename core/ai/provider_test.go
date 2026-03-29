package ai

import "testing"

func TestRepairPrematureTopLevelObjectClosure(t *testing.T) {
	input := `{"analysis_by_section":{"a":"b"}},"background_info":"ok"}`
	got := repairPrematureTopLevelObjectClosure(input)
	want := `{"analysis_by_section":{"a":"b"},"background_info":"ok"}`
	if got != want {
		t.Fatalf("repairPrematureTopLevelObjectClosure() = %q, want %q", got, want)
	}
}

func TestRepairPrematureTopLevelObjectClosureKeepsValidJSON(t *testing.T) {
	input := `{"analysis_by_section":{"a":"b"},"background_info":"ok"}`
	got := repairPrematureTopLevelObjectClosure(input)
	if got != input {
		t.Fatalf("repairPrematureTopLevelObjectClosure() changed valid json: %q", got)
	}
}
