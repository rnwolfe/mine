package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// fakeWriter wraps a bytes.Buffer and lets tests control isTTY behaviour by
// using NewMarkdownWriterWithTTY instead of going through *os.File detection.
// We exercise the exported constructor via the isTTY-override helper below.

// newMarkdownWriterForTest creates a MarkdownWriter with explicit TTY control,
// bypassing the *os.File check so tests can run without an actual TTY.
func newMarkdownWriterForTest(out io.Writer, raw, isTTY bool) *MarkdownWriter {
	return &MarkdownWriter{
		out:   out,
		raw:   raw,
		isTTY: isTTY,
	}
}

// --- Write behaviour ---

func TestMarkdownWriter_RawMode_PassesThrough(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, true, true) // raw=true, TTY=true
	input := "# Hello\n\nSome **bold** text.\n"
	_, err := io.WriteString(mdw, input)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	// In raw mode, writes pass through directly; no buffering.
	if got := buf.String(); got != input {
		t.Errorf("raw mode: got %q, want %q", got, input)
	}
}

func TestMarkdownWriter_NonTTY_PassesThrough(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, false, false) // raw=false, TTY=false
	input := "## Heading\n\n- item\n"
	_, err := io.WriteString(mdw, input)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Non-TTY: pass through without buffering.
	if got := buf.String(); got != input {
		t.Errorf("non-TTY mode: got %q, want %q", got, input)
	}
}

func TestMarkdownWriter_TTYMode_Buffers(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, false, true) // raw=false, TTY=true
	input := "# Title\n\nContent here.\n"
	_, err := io.WriteString(mdw, input)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	// TTY mode buffers — nothing written to underlying writer yet.
	if buf.Len() != 0 {
		t.Errorf("TTY mode should buffer; got %q written before Flush", buf.String())
	}
}

// --- Flush behaviour ---

func TestMarkdownWriter_Flush_RawMode_NoOp(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, true, true)
	io.WriteString(mdw, "data") //nolint:errcheck
	before := buf.Len()
	if err := mdw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	// raw mode: Flush is a no-op (data was already written directly).
	if buf.Len() != before {
		t.Errorf("Flush in raw mode should not write additional bytes")
	}
}

func TestMarkdownWriter_Flush_NonTTY_NoOp(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, false, false)
	io.WriteString(mdw, "data") //nolint:errcheck
	before := buf.Len()
	if err := mdw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if buf.Len() != before {
		t.Errorf("Flush in non-TTY mode should not write additional bytes")
	}
}

func TestMarkdownWriter_Flush_TTYMode_RendersMarkdown(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, false, true)
	io.WriteString(mdw, "# Hello\n\nWorld.\n") //nolint:errcheck

	if err := mdw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	rendered := buf.String()
	if rendered == "" {
		t.Fatal("Flush should produce non-empty rendered output")
	}
	// Glamour outputs styled text; at minimum "Hello" and "World" should appear.
	if !strings.Contains(rendered, "Hello") {
		t.Errorf("rendered output should contain 'Hello'; got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "World") {
		t.Errorf("rendered output should contain 'World'; got:\n%s", rendered)
	}
}

func TestMarkdownWriter_Flush_TTYMode_EmptyBuffer_NoOp(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, false, true)
	// No writes — Flush should be a no-op.
	if err := mdw.Flush(); err != nil {
		t.Fatalf("Flush on empty buffer: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("Flush on empty buffer should write nothing")
	}
}

// --- Multiple writes (chunk simulation) ---

func TestMarkdownWriter_MultipleChunks_TTY(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, false, true)

	chunks := []string{"# Head", "ing\n\n", "- item 1\n", "- item 2\n"}
	for _, c := range chunks {
		if _, err := io.WriteString(mdw, c); err != nil {
			t.Fatalf("Write chunk %q: %v", c, err)
		}
	}
	// Nothing written before Flush.
	if buf.Len() != 0 {
		t.Errorf("expected nothing before Flush; got %q", buf.String())
	}

	if err := mdw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if !strings.Contains(buf.String(), "item 1") {
		t.Errorf("rendered output should contain list items; got:\n%s", buf.String())
	}
}

func TestMarkdownWriter_MultipleChunks_Raw(t *testing.T) {
	var buf bytes.Buffer
	mdw := newMarkdownWriterForTest(&buf, true, true)

	chunks := []string{"chunk1", " chunk2", " chunk3"}
	for _, c := range chunks {
		io.WriteString(mdw, c) //nolint:errcheck
	}
	// All chunks written immediately in raw mode.
	want := "chunk1 chunk2 chunk3"
	if got := buf.String(); got != want {
		t.Errorf("raw multi-chunk: got %q, want %q", got, want)
	}
}

// --- RenderMarkdown helper ---

func TestRenderMarkdown_ReturnsStyledOutput(t *testing.T) {
	input := "# Title\n\n**Bold** and _italic_.\n"
	out := RenderMarkdown(input)
	// Should not return empty string on success.
	if out == "" {
		t.Fatal("RenderMarkdown returned empty string")
	}
	// The rendered string should still contain the words.
	if !strings.Contains(out, "Title") {
		t.Errorf("rendered output missing 'Title'; got: %q", out)
	}
}

func TestRenderMarkdown_CodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```\n"
	out := RenderMarkdown(input)
	// Code content should appear in output (possibly styled).
	if !strings.Contains(out, "fmt.Println") {
		t.Errorf("rendered output should preserve code content; got: %q", out)
	}
}

// --- NewMarkdownWriter constructor ---

func TestNewMarkdownWriter_NonTTYFile(t *testing.T) {
	// A regular file (not a terminal) should produce raw=false, isTTY=false.
	f, err := os.CreateTemp(t.TempDir(), "mdw-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	mdw := NewMarkdownWriter(f, false)
	if mdw.raw {
		t.Error("expected raw=false")
	}
	if mdw.isTTY {
		t.Error("expected isTTY=false for a regular file")
	}
}

func TestNewMarkdownWriter_RawFlag(t *testing.T) {
	var buf bytes.Buffer
	mdw := NewMarkdownWriter(&buf, true)
	if !mdw.raw {
		t.Error("expected raw=true when raw flag is set")
	}
}

func TestNewMarkdownWriter_NonFileWriter(t *testing.T) {
	// A plain bytes.Buffer is not an *os.File — isTTY must be false.
	var buf bytes.Buffer
	mdw := NewMarkdownWriter(&buf, false)
	if mdw.isTTY {
		t.Error("expected isTTY=false for a non-file writer")
	}
}

// --- IsStdoutTTY ---

func TestIsStdoutTTY_ReturnsBool(t *testing.T) {
	// We can't control whether stdout is a TTY in a test runner, but we can
	// assert the function returns without panicking and produces a bool.
	_ = IsStdoutTTY()
}
