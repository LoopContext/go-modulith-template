package notifier_test

import (
	"strings"
	"testing"
	"time"

	"github.com/cmelgarejo/go-modulith-template/internal/notifier"
)

func TestNewTemplateManager(t *testing.T) {
	t.Parallel()

	tm := notifier.NewTemplateManager()
	if tm == nil {
		t.Fatal("expected template manager to be created")
	}
}

func TestTemplateManager_RenderHTML_MagicCode(t *testing.T) {
	t.Parallel()

	tm := notifier.NewTemplateManager()

	data := notifier.TemplateData{
		AppName:     "TestApp",
		CompanyName: "Test Company",
		Year:        time.Now().Year(),
		Code:        "123456",
		ExpiresIn:   "15 minutes",
	}

	result, err := tm.RenderHTML("magic_code_email", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "123456") {
		t.Error("expected result to contain the code")
	}

	if !strings.Contains(result, "TestApp") {
		t.Error("expected result to contain app name")
	}

	if !strings.Contains(result, "15 minutes") {
		t.Error("expected result to contain expiration time")
	}
}

func TestTemplateManager_RenderText_MagicCode(t *testing.T) {
	t.Parallel()

	tm := notifier.NewTemplateManager()

	data := notifier.TemplateData{
		AppName:     "TestApp",
		CompanyName: "Test Company",
		Year:        time.Now().Year(),
		Code:        "654321",
		ExpiresIn:   "10 minutes",
	}

	result, err := tm.RenderText("magic_code_email", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "654321") {
		t.Error("expected result to contain the code")
	}
}

func TestTemplateManager_RenderText_SMS(t *testing.T) {
	t.Parallel()

	tm := notifier.NewTemplateManager()

	data := notifier.TemplateData{
		AppName:   "TestApp",
		Code:      "789012",
		ExpiresIn: "5 minutes",
	}

	result, err := tm.RenderText("magic_code_sms", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "789012") {
		t.Error("expected result to contain the code")
	}

	if !strings.Contains(result, "TestApp") {
		t.Error("expected result to contain app name")
	}
}

func TestTemplateManager_RenderHTML_WelcomeEmail(t *testing.T) {
	t.Parallel()

	tm := notifier.NewTemplateManager()

	data := notifier.TemplateData{
		AppName:     "TestApp",
		CompanyName: "Test Company",
		Year:        time.Now().Year(),
		UserName:    "John Doe",
		ActionURL:   "https://example.com/start",
		ActionLabel: "Get Started",
	}

	result, err := tm.RenderHTML("welcome_email", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "John Doe") {
		t.Error("expected result to contain user name")
	}

	if !strings.Contains(result, "https://example.com/start") {
		t.Error("expected result to contain action URL")
	}
}

func TestTemplateManager_AddCustomTemplate(t *testing.T) {
	t.Parallel()

	tm := notifier.NewTemplateManager()

	err := tm.AddHTMLTemplate("custom", "<h1>Hello, {{.UserName}}!</h1>")
	if err != nil {
		t.Fatalf("unexpected error adding template: %v", err)
	}

	data := notifier.TemplateData{UserName: "Alice"}

	result, err := tm.RenderHTML("custom", data)
	if err != nil {
		t.Fatalf("unexpected error rendering: %v", err)
	}

	if result != "<h1>Hello, Alice!</h1>" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestTemplateManager_RenderHTML_NonexistentTemplate(t *testing.T) {
	t.Parallel()

	tm := notifier.NewTemplateManager()

	_, err := tm.RenderHTML("nonexistent", notifier.TemplateData{})
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}

func TestTemplateManager_RenderText_NonexistentTemplate(t *testing.T) {
	t.Parallel()

	tm := notifier.NewTemplateManager()

	_, err := tm.RenderText("nonexistent", notifier.TemplateData{})
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}
