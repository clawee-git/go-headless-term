package headlessterm

import (
	"bytes"
	"maps"
	"sync"
	"testing"
)

// userVarsCases: the user-variable store is set, optionally cleared, and then
// read back. The calls and the expected reads are the only things that vary.
var userVarsCases = []struct {
	name     string
	sets     [][2]string       // SetUserVar calls, in order
	clear    bool              // ClearUserVars after the sets
	wantVars [][2]string       // GetUserVar(name) must return value
	wantAll  map[string]string // GetUserVars() whole; nil means unchecked
}{
	{
		name:     "set and get",
		sets:     [][2]string{{"SANETTY_USER", "daniel"}},
		wantVars: [][2]string{{"SANETTY_USER", "daniel"}},
	},
	{
		name:     "unset returns empty",
		wantVars: [][2]string{{"NONEXISTENT", ""}},
	},
	{
		name:    "get all variables",
		sets:    [][2]string{{"VAR1", "value1"}, {"VAR2", "value2"}, {"VAR3", "value3"}},
		wantAll: map[string]string{"VAR1": "value1", "VAR2": "value2", "VAR3": "value3"},
	},
	{
		name:     "clear all variables",
		sets:     [][2]string{{"VAR1", "value1"}, {"VAR2", "value2"}},
		clear:    true,
		wantVars: [][2]string{{"VAR1", ""}},
		wantAll:  map[string]string{},
	},
	{
		name:     "overwrite",
		sets:     [][2]string{{"VAR1", "initial"}, {"VAR1", "updated"}},
		wantVars: [][2]string{{"VAR1", "updated"}},
	},
	{
		name:     "empty value exists",
		sets:     [][2]string{{"VAR1", ""}},
		wantVars: [][2]string{{"VAR1", ""}},
		wantAll:  map[string]string{"VAR1": ""},
	},
}

func TestUserVars(t *testing.T) {
	for _, c := range userVarsCases {
		t.Run(c.name, func(t *testing.T) {
			term := New()
			for _, set := range c.sets {
				term.SetUserVar(set[0], set[1])
			}
			if c.clear {
				term.ClearUserVars()
			}

			for _, want := range c.wantVars {
				if got := term.GetUserVar(want[0]); got != want[1] {
					t.Errorf("GetUserVar(%q) = %q, want %q", want[0], got, want[1])
				}
			}
			if c.wantAll != nil {
				if got := term.GetUserVars(); !maps.Equal(got, c.wantAll) {
					t.Errorf("GetUserVars() = %v, want %v", got, c.wantAll)
				}
			}
		})
	}

	// The copy contract is behaviour, not data: the caller mutates what
	// GetUserVars returned and the store must not see it.
	t.Run("get all returns a copy", func(t *testing.T) {
		term := New()
		term.SetUserVar("VAR1", "value1")

		vars := term.GetUserVars()
		vars["VAR1"] = "modified"
		vars["NEW_VAR"] = "new_value"

		if got := term.GetUserVar("VAR1"); got != "value1" {
			t.Errorf("GetUserVar(%q) = %q, want %q", "VAR1", got, "value1")
		}
		if got := term.GetUserVar("NEW_VAR"); got != "" {
			t.Errorf("GetUserVar(%q) = %q, want empty", "NEW_VAR", got)
		}
	})
}

func TestUserVarMiddleware(t *testing.T) {
	t.Run("intercepts", func(t *testing.T) {
		middlewareCalled := false
		var interceptedName, interceptedValue string

		term := New(WithMiddleware(&Middleware{
			SetUserVar: func(name, value string, next func(string, string)) {
				middlewareCalled = true
				interceptedName = name
				interceptedValue = value
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
		if got := term.GetUserVar("MODIFIED_VAR1"); got != "MODIFIED_value1" {
			t.Errorf("expected 'MODIFIED_value1', got %q", got)
		}
	})

	t.Run("blocks", func(t *testing.T) {
		term := New(WithMiddleware(&Middleware{
			SetUserVar: func(name, value string, next func(string, string)) {
				// Don't call next - block the operation
			},
		}))

		term.SetUserVar("VAR1", "value1")

		if got := term.GetUserVar("VAR1"); got != "" {
			t.Errorf("expected variable to be blocked, got %q", got)
		}
	})
}

func TestUserVarThreadSafety(t *testing.T) {
	term := New()

	var wg sync.WaitGroup
	const numGoroutines = 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			term.SetUserVar("VAR", "value")
		}()
	}
	wg.Wait()

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = term.GetUserVar("VAR")
			_ = term.GetUserVars()
		}()
	}
	wg.Wait()

	wg.Add(numGoroutines * 2)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			term.SetUserVar("VAR", "value")
		}()
		go func() {
			defer wg.Done()
			_ = term.GetUserVar("VAR")
		}()
	}
	wg.Wait()

	if got := term.GetUserVar("VAR"); got != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}

var osc1337SetUserVarCases = []struct {
	name     string
	osc      string
	varName  string
	expected string
	exists   bool
}{
	{
		name:     "basic BEL terminator",
		osc:      "\x1b]1337;SetUserVar=TEST_VAR=dGVzdF92YWx1ZQ==\x07",
		varName:  "TEST_VAR",
		expected: "test_value",
		exists:   true,
	},
	{
		name:     "ST terminator",
		osc:      "\x1b]1337;SetUserVar=HELLO=aGVsbG8=\x1b\\",
		varName:  "HELLO",
		expected: "hello",
		exists:   true,
	},
	{
		name:    "invalid base64",
		osc:     "\x1b]1337;SetUserVar=TEST=!@#$%^\x07",
		varName: "TEST",
		exists:  false,
	},
	{
		name:     "empty value",
		osc:      "\x1b]1337;SetUserVar=EMPTY=\x07",
		varName:  "EMPTY",
		expected: "",
		exists:   true,
	},
	{
		name:     "special characters",
		osc:      "\x1b]1337;SetUserVar=SPECIAL=aGVsbG8Kd29ybGQJdGFi\x07",
		varName:  "SPECIAL",
		expected: "hello\nworld\ttab",
		exists:   true,
	},
}

func TestOSC1337SetUserVar(t *testing.T) {
	for _, c := range osc1337SetUserVarCases {
		t.Run(c.name, func(t *testing.T) {
			term := New()
			_, _ = term.Write([]byte(c.osc))
			got := term.GetUserVar(c.varName)
			if !c.exists {
				if got != "" {
					t.Errorf("expected empty string for invalid base64, got %q", got)
				}
				return
			}
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}

	t.Run("no response written", func(t *testing.T) {
		var buf bytes.Buffer
		term := New(WithPTYWriter(&buf))
		_, _ = term.Write([]byte("\x1b]1337;SetUserVar=TEST=dGVzdA==\x07"))
		if buf.Len() != 0 {
			t.Errorf("expected no response, got %d bytes", buf.Len())
		}
		if got := term.GetUserVar("TEST"); got != "test" {
			t.Errorf("expected 'test', got %q", got)
		}
	})
}
