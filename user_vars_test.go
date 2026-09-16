package headlessterm

import (
	"bytes"
	"sync"
	"testing"
)

var userVarsCases = []struct {
	name string
	test func(t *testing.T, term *Terminal)
}{
	{
		name: "set and get",
		test: func(t *testing.T, term *Terminal) {
			term.SetUserVar("SANETTY_USER", "daniel")
			if got := term.GetUserVar("SANETTY_USER"); got != "daniel" {
				t.Errorf("expected 'daniel', got %q", got)
			}
		},
	},
	{
		name: "unset returns empty",
		test: func(t *testing.T, term *Terminal) {
			if got := term.GetUserVar("NONEXISTENT"); got != "" {
				t.Errorf("expected empty string for unset variable, got %q", got)
			}
		},
	},
	{
		name: "get all variables",
		test: func(t *testing.T, term *Terminal) {
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
		},
	},
	{
		name: "get all returns a copy",
		test: func(t *testing.T, term *Terminal) {
			term.SetUserVar("VAR1", "value1")
			vars := term.GetUserVars()
			vars["VAR1"] = "modified"
			vars["NEW_VAR"] = "new_value"
			if got := term.GetUserVar("VAR1"); got != "value1" {
				t.Errorf("expected original value 'value1', got %q", got)
			}
			if got := term.GetUserVar("NEW_VAR"); got != "" {
				t.Errorf("expected NEW_VAR to not exist, got %q", got)
			}
		},
	},
	{
		name: "clear all variables",
		test: func(t *testing.T, term *Terminal) {
			term.SetUserVar("VAR1", "value1")
			term.SetUserVar("VAR2", "value2")
			term.ClearUserVars()
			if len(term.GetUserVars()) != 0 {
				t.Errorf("expected 0 variables after clear, got %d", len(term.GetUserVars()))
			}
			if got := term.GetUserVar("VAR1"); got != "" {
				t.Errorf("expected empty string after clear, got %q", got)
			}
		},
	},
	{
		name: "overwrite",
		test: func(t *testing.T, term *Terminal) {
			term.SetUserVar("VAR1", "initial")
			term.SetUserVar("VAR1", "updated")
			if got := term.GetUserVar("VAR1"); got != "updated" {
				t.Errorf("expected 'updated', got %q", got)
			}
		},
	},
	{
		name: "empty value exists",
		test: func(t *testing.T, term *Terminal) {
			term.SetUserVar("VAR1", "")
			if got := term.GetUserVar("VAR1"); got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			vars := term.GetUserVars()
			if _, exists := vars["VAR1"]; !exists {
				t.Error("expected VAR1 to exist with empty value")
			}
		},
	},
}

func TestUserVars(t *testing.T) {
	for _, c := range userVarsCases {
		t.Run(c.name, func(t *testing.T) {
			c.test(t, New())
		})
	}
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

func TestMiddlewareMergeSetUserVar(t *testing.T) {
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

	if !call2 {
		t.Error("SetUserVar middleware should be called after merge")
	}
	if got := term.GetUserVar("TEST"); got != "value" {
		t.Errorf("expected 'value', got %q", got)
	}
}
