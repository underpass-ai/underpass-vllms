package config

import (
	"testing"
	"time"
)

func TestValidationHelpers(t *testing.T) {
	if err := validateModelType(ModelTypeGemma4); err != nil {
		t.Fatalf("unexpected model type error: %v", err)
	}
	if err := validateModelType("bad"); err == nil {
		t.Fatalf("expected invalid model type error")
	}
	if err := validateProvider("PASS2_PROVIDER", "vllm_chat_completions"); err != nil {
		t.Fatalf("unexpected provider error: %v", err)
	}
	if err := validateProvider("PASS2_PROVIDER", "bad"); err == nil {
		t.Fatalf("expected invalid provider error")
	}
}

func TestLookupHelpers(t *testing.T) {
	t.Setenv("REQ_STR", "value")
	t.Setenv("OPT_STR", "")
	t.Setenv("REQ_INT", "10")
	t.Setenv("OPT_INT", "11")
	t.Setenv("REQ_FLOAT", "1.5")
	t.Setenv("OPT_FLOAT", "2.5")
	t.Setenv("REQ_BOOL", "true")
	t.Setenv("OPT_BOOL", "false")
	t.Setenv("REQ_DURATION", "30s")

	if got, err := lookupRequiredString("REQ_STR"); err != nil || got != "value" {
		t.Fatalf("unexpected required string result: %q err=%v", got, err)
	}
	if got, err := lookupOptionalString("OPT_STR"); err != nil || got != nil {
		t.Fatalf("unexpected optional string result: %#v err=%v", got, err)
	}
	if got, err := lookupRequiredInt("REQ_INT"); err != nil || got != 10 {
		t.Fatalf("unexpected required int result: %d err=%v", got, err)
	}
	if got, err := lookupOptionalInt("OPT_INT"); err != nil || got == nil || *got != 11 {
		t.Fatalf("unexpected optional int result: %#v err=%v", got, err)
	}
	if got, err := lookupRequiredFloat("REQ_FLOAT"); err != nil || got != 1.5 {
		t.Fatalf("unexpected required float result: %v err=%v", got, err)
	}
	if got, err := lookupOptionalFloat("OPT_FLOAT"); err != nil || got == nil || *got != 2.5 {
		t.Fatalf("unexpected optional float result: %#v err=%v", got, err)
	}
	if got, err := lookupRequiredBool("REQ_BOOL"); err != nil || got != true {
		t.Fatalf("unexpected required bool result: %v err=%v", got, err)
	}
	if got, err := lookupOptionalBool("OPT_BOOL"); err != nil || got == nil || *got != false {
		t.Fatalf("unexpected optional bool result: %#v err=%v", got, err)
	}
	if got, err := lookupRequiredDuration("REQ_DURATION"); err != nil || got != 30*time.Second {
		t.Fatalf("unexpected duration result: %v err=%v", got, err)
	}

	if got, err := lookupRequiredOptionalFloat("REQ_FLOAT"); err != nil || got == nil || *got != 1.5 {
		t.Fatalf("unexpected required optional float: %#v err=%v", got, err)
	}
	if got, err := lookupRequiredOptionalInt("REQ_INT"); err != nil || got == nil || *got != 10 {
		t.Fatalf("unexpected required optional int: %#v err=%v", got, err)
	}
	if got, err := lookupRequiredOptionalBool("REQ_BOOL"); err != nil || got == nil || *got != true {
		t.Fatalf("unexpected required optional bool: %#v err=%v", got, err)
	}
}

func TestLookupHelpersReturnErrors(t *testing.T) {
	t.Setenv("BAD_INT", "x")
	t.Setenv("BAD_FLOAT", "x")
	t.Setenv("BAD_BOOL", "x")
	t.Setenv("BAD_DURATION", "x")

	if _, err := lookupRequiredString("MISSING_STR"); err == nil {
		t.Fatalf("expected missing string error")
	}
	if _, err := lookupRequiredInt("BAD_INT"); err == nil {
		t.Fatalf("expected bad int error")
	}
	if _, err := lookupOptionalInt("BAD_INT"); err == nil {
		t.Fatalf("expected bad optional int error")
	}
	if _, err := lookupRequiredFloat("BAD_FLOAT"); err == nil {
		t.Fatalf("expected bad float error")
	}
	if _, err := lookupOptionalFloat("BAD_FLOAT"); err == nil {
		t.Fatalf("expected bad optional float error")
	}
	if _, err := lookupRequiredBool("BAD_BOOL"); err == nil {
		t.Fatalf("expected bad bool error")
	}
	if _, err := lookupOptionalBool("BAD_BOOL"); err == nil {
		t.Fatalf("expected bad optional bool error")
	}
	if _, err := lookupRequiredDuration("BAD_DURATION"); err == nil {
		t.Fatalf("expected bad duration error")
	}
}
