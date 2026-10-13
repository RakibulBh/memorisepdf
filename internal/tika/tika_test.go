package tika

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExtractSendsThePutTikaExpects(t *testing.T) {
	var gotMethod, gotPath, gotAccept, gotContentType string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = readAll(r)

		w.Write([]byte("  extracted text  \n"))
	}))
	defer server.Close()

	text, err := New(server.URL, 5*time.Second).
		Extract(context.Background(), strings.NewReader("file bytes"), "application/pdf")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	// Trimmed, because Tika pads its plain-text output with whitespace.
	if text != "extracted text" {
		t.Errorf("text = %q, want %q", text, "extracted text")
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if gotPath != "/tika" {
		t.Errorf("path = %s, want /tika", gotPath)
	}
	if gotAccept != "text/plain" {
		t.Errorf("Accept = %q, want text/plain", gotAccept)
	}
	if gotContentType != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", gotContentType)
	}
	if string(gotBody) != "file bytes" {
		t.Errorf("body = %q, want the file bytes", gotBody)
	}
}

// A 422 means the file itself is the problem, which the caller distinguishes
// from the server being broken.
func TestExtractReportsUnsupportedSeparately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("cannot parse"))
	}))
	defer server.Close()

	_, err := New(server.URL, 5*time.Second).
		Extract(context.Background(), strings.NewReader("x"), "application/pdf")

	if err == nil {
		t.Fatal("Extract succeeded on a 422, want an error")
	}
	if !strings.Contains(err.Error(), ErrUnsupported.Error()) {
		t.Errorf("err = %v, want it to wrap ErrUnsupported", err)
	}
}

func TestExtractReportsServerErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := New(server.URL, 5*time.Second).
		Extract(context.Background(), strings.NewReader("x"), "")
	if err == nil {
		t.Fatal("Extract succeeded on a 500, want an error")
	}
	if strings.Contains(err.Error(), ErrUnsupported.Error()) {
		t.Errorf("err = %v, should not be reported as an unsupported document", err)
	}
}

// A trailing slash on the configured URL must not produce //tika.
func TestNewTrimsTrailingSlash(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	if _, err := New(server.URL+"/", 5*time.Second).
		Extract(context.Background(), strings.NewReader("x"), ""); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if gotPath != "/tika" {
		t.Errorf("path = %q, want /tika", gotPath)
	}
}

// Runs against a real Tika, which is the only way to know the request this
// package builds is one Tika actually accepts. Start one with `make dev-up`:
//
//	TEST_TIKA_URL=http://localhost:9998 go test ./internal/tika
func TestExtractAgainstRealTika(t *testing.T) {
	url := os.Getenv("TEST_TIKA_URL")
	if url == "" {
		t.Skip("set TEST_TIKA_URL to run this against a real Tika server")
	}

	client := New(url, 60*time.Second)
	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	pdf := os.Getenv("TEST_PDF")
	if pdf == "" {
		t.Skip("set TEST_PDF to a real pdf to check extraction")
	}

	body, err := os.ReadFile(pdf)
	if err != nil {
		t.Fatalf("read test pdf: %v", err)
	}

	text, err := client.Extract(ctx, bytes.NewReader(body), "application/pdf")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	for _, want := range []string{"Normalisation", "atomic", "transitive"} {
		if !strings.Contains(text, want) {
			t.Errorf("extracted text is missing %q; got:\n%s", want, text)
		}
	}
}

// Reads a request body in full.
func readAll(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	return buf.Bytes(), err
}
