package headlessterm

import (
	"bytes"
	"testing"
)

// testNotificationProvider is a test implementation of NotificationProvider
type testNotificationProvider struct {
	payloads     []*NotificationPayload
	queryReply   string
	notifyCount  int
}

func (p *testNotificationProvider) Notify(payload *NotificationPayload) string {
	p.notifyCount++
	p.payloads = append(p.payloads, payload)

	// Return query response if this is a query request
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

// TestNoopNotification tests that NoopNotification implements NotificationProvider
func TestNoopNotification(t *testing.T) {
	var provider NotificationProvider = NoopNotification{}

	payload := &NotificationPayload{
		PayloadType: "title",
		Data:        []byte("Test"),
	}

	// Should return empty string and not panic
	response := provider.Notify(payload)
	if response != "" {
		t.Errorf("expected empty response from NoopNotification, got %q", response)
	}
}

// TestWithNotificationOption tests the WithNotification option
func TestWithNotificationOption(t *testing.T) {
	provider := &testNotificationProvider{}
	term := New(WithNotification(provider))

	if term.NotificationProvider() != provider {
		t.Error("expected custom notification provider to be set")
	}
}

// TestDefaultNotificationProvider tests that default provider is NoopNotification
func TestDefaultNotificationProvider(t *testing.T) {
	term := New()

	provider := term.NotificationProvider()
	if provider == nil {
		t.Fatal("expected default notification provider to be set")
	}

	// Should be NoopNotification (returns empty string)
	payload := &NotificationPayload{PayloadType: "title", Data: []byte("Test")}
	response := provider.Notify(payload)
	if response != "" {
		t.Errorf("expected empty response from default provider, got %q", response)
	}
}

// TestSetNotificationProvider tests setting the provider at runtime
func TestSetNotificationProvider(t *testing.T) {
	term := New()
	provider := &testNotificationProvider{}

	term.SetNotificationProvider(provider)

	if term.NotificationProvider() != provider {
		t.Error("expected notification provider to be updated")
	}
}

// TestDesktopNotificationHandler tests the DesktopNotification handler
func TestDesktopNotificationHandler(t *testing.T) {
	provider := &testNotificationProvider{}
	term := New(WithNotification(provider))

	payload := &NotificationPayload{
		ID:          "test-1",
		PayloadType: "title",
		Data:        []byte("Test Title"),
		Done:        true,
	}

	term.DesktopNotification(payload)

	if provider.notifyCount != 1 {
		t.Errorf("expected 1 notification, got %d", provider.notifyCount)
	}

	last := provider.LastPayload()
	if last == nil {
		t.Fatal("expected payload to be recorded")
	}
	if last.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got %q", last.ID)
	}
	if string(last.Data) != "Test Title" {
		t.Errorf("expected data 'Test Title', got %q", string(last.Data))
	}
}

// TestDesktopNotificationWithNilProvider tests handler with nil provider
func TestDesktopNotificationWithNilProvider(t *testing.T) {
	term := New()
	term.SetNotificationProvider(nil)

	payload := &NotificationPayload{
		PayloadType: "title",
		Data:        []byte("Test"),
	}

	// Should not panic with nil provider
	term.DesktopNotification(payload)
}

// TestDesktopNotificationQueryResponse tests query response writing
func TestDesktopNotificationQueryResponse(t *testing.T) {
	var responses []byte
	writer := &bytes.Buffer{}

	provider := &testNotificationProvider{
		queryReply: "\x1b]99;i=test;p=?\x1b\\",
	}

	term := New(
		WithNotification(provider),
		WithResponse(writer),
	)

	payload := &NotificationPayload{
		ID:          "test",
		PayloadType: "?",
		Done:        true,
	}

	term.DesktopNotification(payload)

	responses = writer.Bytes()
	if len(responses) == 0 {
		t.Error("expected query response to be written")
	}
	if string(responses) != provider.queryReply {
		t.Errorf("expected response %q, got %q", provider.queryReply, string(responses))
	}
}

// TestDesktopNotificationMiddleware tests middleware interception
func TestDesktopNotificationMiddleware(t *testing.T) {
	provider := &testNotificationProvider{}
	middlewareCalled := false
	var interceptedPayload *NotificationPayload

	term := New(
		WithNotification(provider),
		WithMiddleware(&Middleware{
			DesktopNotification: func(payload *NotificationPayload, next func(*NotificationPayload)) {
				middlewareCalled = true
				interceptedPayload = payload
				// Modify payload before passing to provider
				modifiedPayload := *payload
				modifiedPayload.ID = "modified-" + payload.ID
				next(&modifiedPayload)
			},
		}),
	)

	payload := &NotificationPayload{
		ID:          "original",
		PayloadType: "title",
		Data:        []byte("Test"),
	}

	term.DesktopNotification(payload)

	if !middlewareCalled {
		t.Error("expected middleware to be called")
	}
	if interceptedPayload == nil || interceptedPayload.ID != "original" {
		t.Error("expected middleware to receive original payload")
	}
	if provider.notifyCount != 1 {
		t.Errorf("expected 1 notification, got %d", provider.notifyCount)
	}

	// Provider should receive modified payload
	last := provider.LastPayload()
	if last.ID != "modified-original" {
		t.Errorf("expected modified ID 'modified-original', got %q", last.ID)
	}
}

// TestDesktopNotificationMiddlewareBlocks tests middleware blocking notifications
func TestDesktopNotificationMiddlewareBlocks(t *testing.T) {
	provider := &testNotificationProvider{}

	term := New(
		WithNotification(provider),
		WithMiddleware(&Middleware{
			DesktopNotification: func(payload *NotificationPayload, next func(*NotificationPayload)) {
				// Don't call next - block the notification
			},
		}),
	)

	payload := &NotificationPayload{
		PayloadType: "title",
		Data:        []byte("Test"),
	}

	term.DesktopNotification(payload)

	if provider.notifyCount != 0 {
		t.Errorf("expected 0 notifications (blocked by middleware), got %d", provider.notifyCount)
	}
}

// TestNotificationPayloadFields tests that all payload fields are accessible
func TestNotificationPayloadFields(t *testing.T) {
	provider := &testNotificationProvider{}
	term := New(WithNotification(provider))

	payload := &NotificationPayload{
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
	}

	term.DesktopNotification(payload)

	last := provider.LastPayload()
	if last == nil {
		t.Fatal("expected payload to be recorded")
	}

	// Verify all fields
	if last.ID != "notify-123" {
		t.Errorf("ID mismatch: %q", last.ID)
	}
	if !last.Done {
		t.Error("Done should be true")
	}
	if last.PayloadType != "body" {
		t.Errorf("PayloadType mismatch: %q", last.PayloadType)
	}
	if last.Encoding != "1" {
		t.Errorf("Encoding mismatch: %q", last.Encoding)
	}
	if len(last.Actions) != 2 || last.Actions[0] != "focus" {
		t.Errorf("Actions mismatch: %v", last.Actions)
	}
	if !last.TrackClose {
		t.Error("TrackClose should be true")
	}
	if last.Timeout != 5000 {
		t.Errorf("Timeout mismatch: %d", last.Timeout)
	}
	if last.AppName != "TestApp" {
		t.Errorf("AppName mismatch: %q", last.AppName)
	}
	if last.Type != "alert" {
		t.Errorf("Type mismatch: %q", last.Type)
	}
	if last.IconName != "warning" {
		t.Errorf("IconName mismatch: %q", last.IconName)
	}
	if last.IconCacheID != "cache-456" {
		t.Errorf("IconCacheID mismatch: %q", last.IconCacheID)
	}
	if last.Sound != "system" {
		t.Errorf("Sound mismatch: %q", last.Sound)
	}
	if last.Urgency != 2 {
		t.Errorf("Urgency mismatch: %d", last.Urgency)
	}
	if last.Occasion != "always" {
		t.Errorf("Occasion mismatch: %q", last.Occasion)
	}
	if string(last.Data) != "Notification body content" {
		t.Errorf("Data mismatch: %q", string(last.Data))
	}
}

// TestMiddlewareMergeDesktopNotification tests that Middleware.Merge includes DesktopNotification
func TestMiddlewareMergeDesktopNotification(t *testing.T) {
	notifyCount := 0

	mw1 := &Middleware{
		Bell: func(next func()) {
			next()
		},
	}

	mw2 := &Middleware{
		DesktopNotification: func(payload *NotificationPayload, next func(*NotificationPayload)) {
			notifyCount++
			next(payload)
		},
	}

	mw1.Merge(mw2)

	provider := &testNotificationProvider{}
	term := New(
		WithNotification(provider),
		WithMiddleware(mw1),
	)

	payload := &NotificationPayload{
		PayloadType: "title",
		Data:        []byte("Test"),
	}

	term.DesktopNotification(payload)

	if notifyCount != 1 {
		t.Errorf("expected 1 middleware call after merge, got %d", notifyCount)
	}
	if provider.notifyCount != 1 {
		t.Errorf("expected 1 provider call, got %d", provider.notifyCount)
	}
}

// TestNotificationProviderThreadSafety tests concurrent access to notification provider
func TestNotificationProviderThreadSafety(t *testing.T) {
	provider := &testNotificationProvider{}
	term := New(WithNotification(provider))

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			payload := &NotificationPayload{
				ID:          "test",
				PayloadType: "title",
				Data:        []byte("Test"),
			}
			term.DesktopNotification(payload)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have received notifications
	if provider.notifyCount != 10 {
		t.Errorf("expected 10 notifications, got %d", provider.notifyCount)
	}
}

// TestNotificationEmptyPayload tests handling of empty payload
func TestNotificationEmptyPayload(t *testing.T) {
	provider := &testNotificationProvider{}
	term := New(WithNotification(provider))

	// Empty payload
	payload := &NotificationPayload{}

	// Should not panic
	term.DesktopNotification(payload)

	if provider.notifyCount != 1 {
		t.Errorf("expected 1 notification, got %d", provider.notifyCount)
	}
}
