package headlessterm

import (
	"bytes"
	"sync"
	"testing"
)

// TestSetUserVar tests setting a user variable
func TestSetUserVar(t *testing.T) {
	term := New()

	term.SetUserVar("SANETTY_USER", "daniel")

	if val := term.GetUserVar("SANETTY_USER"); val != "daniel" {
		t.Errorf("expected 'daniel', got %q", val)
	}
}

// TestGetUserVarNotSet tests getting a user variable that was not set
func TestGetUserVarNotSet(t *testing.T) {
	term := New()

	if val := term.GetUserVar("NONEXISTENT"); val != "" {
		t.Errorf("expected empty string for unset variable, got %q", val)
	}
}

// TestGetUserVars tests getting all user variables
func TestGetUserVars(t *testing.T) {
	term := New()

	term.SetUserVar("VAR1", "value1")
	term.SetUserVar("VAR2", "value2")
	term.SetUserVar("VAR3", "value3")

	vars := term.GetUserVars()

	if len(vars) != 3 {
		t.Errorf("expected 3 variables, got %d", len(vars))
	}
	if vars["VAR1"] != "value1" {
		t.Errorf("VAR1: expected 'value1', got %q", vars["VAR1"])
	}
	if vars["VAR2"] != "value2" {
		t.Errorf("VAR2: expected 'value2', got %q", vars["VAR2"])
	}
	if vars["VAR3"] != "value3" {
		t.Errorf("VAR3: expected 'value3', got %q", vars["VAR3"])
	}
}

// TestGetUserVarsReturnsACopy tests that GetUserVars returns a copy
func TestGetUserVarsReturnsACopy(t *testing.T) {
	term := New()

	term.SetUserVar("VAR1", "value1")

	vars := term.GetUserVars()
	vars["VAR1"] = "modified"
	vars["NEW_VAR"] = "new_value"

	// Original should be unchanged
	if val := term.GetUserVar("VAR1"); val != "value1" {
		t.Errorf("expected original value 'value1', got %q", val)
	}
	if val := term.GetUserVar("NEW_VAR"); val != "" {
		t.Errorf("expected NEW_VAR to not exist, got %q", val)
	}
}

// TestClearUserVars tests clearing all user variables
func TestClearUserVars(t *testing.T) {
	term := New()

	term.SetUserVar("VAR1", "value1")
	term.SetUserVar("VAR2", "value2")

	term.ClearUserVars()

	vars := term.GetUserVars()
	if len(vars) != 0 {
		t.Errorf("expected 0 variables after clear, got %d", len(vars))
	}
	if val := term.GetUserVar("VAR1"); val != "" {
		t.Errorf("expected empty string after clear, got %q", val)
	}
}

// TestUserVarOverwrite tests overwriting a user variable
func TestUserVarOverwrite(t *testing.T) {
	term := New()

	term.SetUserVar("VAR1", "initial")
	term.SetUserVar("VAR1", "updated")

	if val := term.GetUserVar("VAR1"); val != "updated" {
		t.Errorf("expected 'updated', got %q", val)
	}
}

// TestUserVarEmptyValue tests setting an empty value
func TestUserVarEmptyValue(t *testing.T) {
	term := New()

	term.SetUserVar("VAR1", "")

	if val := term.GetUserVar("VAR1"); val != "" {
		t.Errorf("expected empty string, got %q", val)
	}

	vars := term.GetUserVars()
	if _, exists := vars["VAR1"]; !exists {
		t.Error("expected VAR1 to exist with empty value")
	}
}

// TestUserVarMiddleware tests middleware interception
func TestUserVarMiddleware(t *testing.T) {
	middlewareCalled := false
	var interceptedName, interceptedValue string

	term := New(WithMiddleware(&Middleware{
		SetUserVar: func(name, value string, next func(string, string)) {
			middlewareCalled = true
			interceptedName = name
			interceptedValue = value
			// Modify before passing to internal
			next("MODIFIED_"+name, "MODIFIED_"+value)
		},
	}))

	term.SetUserVar("VAR1", "value1")

	if !middlewareCalled {
		t.Error("expected middleware to be called")
	}
	if interceptedName != "VAR1" {
		t.Errorf("expected intercepted name 'VAR1', got %q", interceptedName)
	}
	if interceptedValue != "value1" {
		t.Errorf("expected intercepted value 'value1', got %q", interceptedValue)
	}

	// Should have modified name/value
	if val := term.GetUserVar("MODIFIED_VAR1"); val != "MODIFIED_value1" {
		t.Errorf("expected 'MODIFIED_value1', got %q", val)
	}
}

// TestUserVarMiddlewareBlocks tests middleware blocking
func TestUserVarMiddlewareBlocks(t *testing.T) {
	term := New(WithMiddleware(&Middleware{
		SetUserVar: func(name, value string, next func(string, string)) {
			// Don't call next - block the operation
		},
	}))

	term.SetUserVar("VAR1", "value1")

	if val := term.GetUserVar("VAR1"); val != "" {
		t.Errorf("expected variable to be blocked, got %q", val)
	}
}

// TestUserVarThreadSafety tests concurrent access
func TestUserVarThreadSafety(t *testing.T) {
	term := New()

	var wg sync.WaitGroup
	const numGoroutines = 100

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			term.SetUserVar("VAR", "value")
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = term.GetUserVar("VAR")
			_ = term.GetUserVars()
		}()
	}
	wg.Wait()

	// Concurrent mixed reads/writes
	wg.Add(numGoroutines * 2)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			term.SetUserVar("VAR", "value")
		}(i)
		go func() {
			defer wg.Done()
			_ = term.GetUserVar("VAR")
		}()
	}
	wg.Wait()

	// Should not panic and final value should be set
	if val := term.GetUserVar("VAR"); val != "value" {
		t.Errorf("expected 'value', got %q", val)
	}
}

// TestOSC1337SetUserVar tests OSC 1337 sequence parsing
func TestOSC1337SetUserVar(t *testing.T) {
	term := New()

	// OSC 1337 ; SetUserVar=NAME=BASE64_VALUE ST
	// "test_value" in base64 is "dGVzdF92YWx1ZQ=="
	osc := "\x1b]1337;SetUserVar=TEST_VAR=dGVzdF92YWx1ZQ==\x07"

	_, _ = term.Write([]byte(osc))

	if val := term.GetUserVar("TEST_VAR"); val != "test_value" {
		t.Errorf("expected 'test_value', got %q", val)
	}
}

// TestOSC1337SetUserVarWithST tests OSC 1337 with ST terminator
func TestOSC1337SetUserVarWithST(t *testing.T) {
	term := New()

	// Using \x1b\\ as ST terminator
	// "hello" in base64 is "aGVsbG8="
	osc := "\x1b]1337;SetUserVar=HELLO=aGVsbG8=\x1b\\"

	_, _ = term.Write([]byte(osc))

	if val := term.GetUserVar("HELLO"); val != "hello" {
		t.Errorf("expected 'hello', got %q", val)
	}
}

// TestOSC1337InvalidBase64 tests invalid base64 handling
func TestOSC1337InvalidBase64(t *testing.T) {
	term := New()

	// Invalid base64
	osc := "\x1b]1337;SetUserVar=TEST=!@#$%^\x07"

	_, _ = term.Write([]byte(osc))

	// Should not set the variable
	if val := term.GetUserVar("TEST"); val != "" {
		t.Errorf("expected empty string for invalid base64, got %q", val)
	}
}

// TestOSC1337EmptyValue tests empty base64 value
func TestOSC1337EmptyValue(t *testing.T) {
	term := New()

	// Empty string in base64 is ""
	osc := "\x1b]1337;SetUserVar=EMPTY=\x07"

	_, _ = term.Write([]byte(osc))

	// Should set empty value
	vars := term.GetUserVars()
	if _, exists := vars["EMPTY"]; !exists {
		t.Error("expected EMPTY variable to exist")
	}
}

// TestOSC1337SpecialCharacters tests special characters in value
func TestOSC1337SpecialCharacters(t *testing.T) {
	term := New()

	// "hello\nworld\ttab" in base64 is "aGVsbG8Kd29ybGQJdGFi"
	osc := "\x1b]1337;SetUserVar=SPECIAL=aGVsbG8Kd29ybGQJdGFi\x07"

	_, _ = term.Write([]byte(osc))

	expected := "hello\nworld\ttab"
	if val := term.GetUserVar("SPECIAL"); val != expected {
		t.Errorf("expected %q, got %q", expected, val)
	}
}

// TestUserVarsWithResponse tests that OSC 1337 works with response writer
func TestUserVarsWithResponse(t *testing.T) {
	var buf bytes.Buffer
	term := New(WithResponse(&buf))

	// OSC 1337 SetUserVar doesn't generate a response
	osc := "\x1b]1337;SetUserVar=TEST=dGVzdA==\x07"

	_, _ = term.Write([]byte(osc))

	if buf.Len() != 0 {
		t.Errorf("expected no response, got %d bytes", buf.Len())
	}

	// But variable should be set
	if val := term.GetUserVar("TEST"); val != "test" {
		t.Errorf("expected 'test', got %q", val)
	}
}

// TestMiddlewareMergeSetUserVar tests middleware merge for SetUserVar
func TestMiddlewareMergeSetUserVar(t *testing.T) {
	call1 := false
	call2 := false

	mw1 := &Middleware{
		Bell: func(next func()) {
			next()
		},
	}

	mw2 := &Middleware{
		SetUserVar: func(name, value string, next func(string, string)) {
			call2 = true
			next(name, value)
		},
	}

	mw1.Merge(mw2)

	term := New(WithMiddleware(mw1))
	term.SetUserVar("TEST", "value")

	if call1 {
		t.Error("Bell middleware should not be called")
	}
	if !call2 {
		t.Error("SetUserVar middleware should be called after merge")
	}
	if val := term.GetUserVar("TEST"); val != "value" {
		t.Errorf("expected 'value', got %q", val)
	}
}
