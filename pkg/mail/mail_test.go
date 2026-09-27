package mail

import (
	"strings"
	"testing"
	"time"
)

func TestRenderOTP(t *testing.T) {
	html, err := RenderOTP(OTPData{AppName: "Notebook", Code: "123456", ExpiresMinutes: 5})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"123456", "5 minutes", "Notebook"} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered email does not contain %q", want)
		}
	}
}

func TestBuildMessage(t *testing.T) {
	msg := string(buildMessage("Notebook", "no-reply@example.com", "user@example.com",
		"Code\r\nBcc: evil@example.com", "<p>hi</p>\n", time.Unix(0, 0)))

	header, body, ok := strings.Cut(msg, "\r\n\r\n")
	if !ok {
		t.Fatal("no blank line between header and body")
	}
	if strings.Contains(header, "\r\nBcc:") {
		t.Error("subject injected a header")
	}
	for _, want := range []string{
		`From: "Notebook" <no-reply@example.com>`,
		"To: <user@example.com>",
		"Content-Type: text/html; charset=UTF-8",
	} {
		if !strings.Contains(header, want) {
			t.Errorf("header does not contain %q:\n%s", want, header)
		}
	}
	if body != "<p>hi</p>\r\n" {
		t.Errorf("body = %q", body)
	}
}
