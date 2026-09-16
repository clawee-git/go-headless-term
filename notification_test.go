package headlessterm

import (
	"bytes"
	"testing"
)

// testNotificationProvider is a test implementation of NotificationProvider
type testNotificationProvider struct {
	payloads    []*NotificationPayload
	queryReply  string
	notifyCount int
}

func (p *testNotificationProvider) Notify(payload *NotificationPayload) string {
	p.notifyCount++
	p.payloads = append(p.payloads, payload)

	if payload.PayloadType == "?" {
		return p.queryReply
	}
	return ""
}

func (p *testNotificationProvider) LastPayload() *NotificationPayload {
	if len(p.payloads) == 0 {
		return nil
	}
	return p.payloads[len(p.payloads)-1]
}

func (p *testNotificationProvider) Reset() {
	p.payloads = nil
	p.notifyCount = 0
}

func TestNotificationProviderWiring(t *testing.T) {
	t.Run("default is NoopNotification", func(t *testing.T) {
		term := New()
		provider := term.NotificationProvider()
		if provider == nil {
			t.Fatal("expected default notification provider to be set")
		}
		response := provider.Notify(&NotificationPayload{PayloadType: "title", Data: []byte("Test")})
		if response != "" {
			t.Errorf("expected empty response from default provider, got %q", response)
		}
	})

	t.Run("WithNotification option", func(t *testing.T) {
		provider := &testNotificationProvider{}
		term := New(WithNotification(provider))
		if term.NotificationProvider() != provider {
			t.Error("expected custom notification provider to be set")
		}
	})

	t.Run("SetNotificationProvider at runtime", func(t *testing.T) {
		term := New()
		provider := &testNotificationProvider{}
		term.SetNotificationProvider(provider)
		if term.NotificationProvider() != provider {
			t.Error("expected notification provider to be updated")
		}
	})
}

var desktopNotificationCases = []struct {
	name       string
	provider   *testNotificationProvider
	payload    *NotificationPayload
	wantCount  int
	checkLast  bool
	wantID     string
	wantData   string
	wantField  string
	fieldValue interface{}
}{
	{
		name:      "basic handler",
		provider:  &testNotificationProvider{},
		payload:   &NotificationPayload{ID: "test-1", PayloadType: "title", Data: []byte("Test Title"), Done: true},
		wantCount: 1,
		checkLast: true,
		wantID:    "test-1",
		wantData:  "Test Title",
	},
	{
		name:      "nil provider",
		provider:  nil,
		payload:   &NotificationPayload{PayloadType: "title", Data: []byte("Test")},
		wantCount: 0,
	},
	{
		name:     "full payload fields",
		provider: &testNotificationProvider{},
		payload: &NotificationPayload{
			ID:          "notify-123",
			Done:        true,
			PayloadType: "body",
			Encoding:    "1",
			Actions:     []string{"focus", "report"},
			TrackClose:  true,
			Timeout:     5000,
			AppName:     "TestApp",
			Type:        "alert",
			IconName:    "warning",
			IconCacheID: "cache-456",
			Sound:       "system",
			Urgency:     2,
			Occasion:    "always",
			Data:        []byte("Notification body content"),
		},
		wantCount: 1,
		checkLast: true,
		wantID:    "notify-123",
		wantData:  "Notification body content",
	},
	{
		name:      "empty payload",
		provider:  &testNotificationProvider{},
		payload:   &NotificationPayload{},
		wantCount: 1,
	},
}

func TestDesktopNotification_Cases(t *testing.T) {
	for _, c := range desktopNotificationCases {
		t.Run(c.name, func(t *testing.T) {
			term := New()
			if c.provider != nil {
				term.SetNotificationProvider(c.provider)
			} else {
				term.SetNotificationProvider(nil)
			}

			term.DesktopNotification(c.payload)

			if c.provider == nil {
				return
			}
			if c.provider.notifyCount != c.wantCount {
				t.Errorf("expected %d notifications, got %d", c.wantCount, c.provider.notifyCount)
			}
			if !c.checkLast {
				return
			}
			last := c.provider.LastPayload()
			if last == nil {
				t.Fatal("expected payload to be recorded")
			}
			if last.ID != c.wantID {
				t.Errorf("expected ID %q, got %q", c.wantID, last.ID)
			}
			if string(last.Data) != c.wantData {
				t.Errorf("expected data %q, got %q", c.wantData, string(last.Data))
			}
		})
	}
}

func TestDesktopNotificationQueryResponse(t *testing.T) {
	var responses []byte
	writer := &bytes.Buffer{}

	provider := &testNotificationProvider{
		queryReply: "\x1b]99;i=test;p=?\x1b\\",
	}

	term := New(
		WithNotification(provider),
		WithPTYWriter(writer),
	)

	term.DesktopNotification(&NotificationPayload{
		ID:          "test",
		PayloadType: "?",
		Done:        true,
	})

	responses = writer.Bytes()
	if len(responses) == 0 {
		t.Error("expected query response to be written")
	}
	if string(responses) != provider.queryReply {
		t.Errorf("expected response %q, got %q", provider.queryReply, string(responses))
	}
}

func TestDesktopNotificationMiddleware(t *testing.T) {
	t.Run("intercepts and modifies", func(t *testing.T) {
		provider := &testNotificationProvider{}
		middlewareCalled := false
		var interceptedPayload *NotificationPayload

		term := New(
			WithNotification(provider),
			WithMiddleware(&Middleware{
				DesktopNotification: func(payload *NotificationPayload, next func(*NotificationPayload)) {
					middlewareCalled = true
					interceptedPayload = payload
					modifiedPayload := *payload
					modifiedPayload.ID = "modified-" + payload.ID
					next(&modifiedPayload)
				},
			}),
		)

		term.DesktopNotification(&NotificationPayload{ID: "original", PayloadType: "title", Data: []byte("Test")})

		if !middlewareCalled {
			t.Error("expected middleware to be called")
		}
		if interceptedPayload == nil || interceptedPayload.ID != "original" {
			t.Error("expected middleware to receive original payload")
		}
		if provider.notifyCount != 1 {
			t.Errorf("expected 1 notification, got %d", provider.notifyCount)
		}
		last := provider.LastPayload()
		if last.ID != "modified-original" {
			t.Errorf("expected modified ID 'modified-original', got %q", last.ID)
		}
	})

	t.Run("blocks", func(t *testing.T) {
		provider := &testNotificationProvider{}

		term := New(
			WithNotification(provider),
			WithMiddleware(&Middleware{
				DesktopNotification: func(payload *NotificationPayload, next func(*NotificationPayload)) {
					// Don't call next - block the notification
				},
			}),
		)

		term.DesktopNotification(&NotificationPayload{PayloadType: "title", Data: []byte("Test")})

		if provider.notifyCount != 0 {
			t.Errorf("expected 0 notifications (blocked by middleware), got %d", provider.notifyCount)
		}
	})
}

func TestNotificationProviderThreadSafety(t *testing.T) {
	provider := &testNotificationProvider{}
	term := New(WithNotification(provider))

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			term.DesktopNotification(&NotificationPayload{ID: "test", PayloadType: "title", Data: []byte("Test")})
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if provider.notifyCount != 10 {
		t.Errorf("expected 10 notifications, got %d", provider.notifyCount)
	}
}
