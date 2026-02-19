package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/mattn/go-isatty"
)

// IsStdoutTTY returns true when stdout is connected to a terminal.
func IsStdoutTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// MarkdownWriter is an io.Writer that buffers streamed content and renders it
// as styled terminal markdown (via glamour) when Flush is called.
//
// In raw mode or non-TTY contexts, all writes pass through immediately to the
// underlying writer without buffering. Call Flush after the streaming source
// closes to emit the rendered output.
type MarkdownWriter struct {
	out   io.Writer
	buf   bytes.Buffer
	raw   bool // --raw flag: force plain output regardless of TTY
	isTTY bool // whether the underlying writer is a terminal
}

// NewMarkdownWriter creates a MarkdownWriter targeting out.
//
//   - raw=true  → plain pass-through (no buffering, no rendering)
//   - out is a non-TTY *os.File → plain pass-through
//   - out is a TTY *os.File     → buffer chunks, render on Flush
func NewMarkdownWriter(out io.Writer, raw bool) *MarkdownWriter {
	tty := false
	if f, ok := out.(*os.File); ok {
		tty = isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return &MarkdownWriter{
		out:   out,
		raw:   raw,
		isTTY: tty,
	}
}

// Write satisfies io.Writer. In render mode the data is buffered; in raw/non-TTY
// mode it is forwarded directly to the underlying writer.
func (m *MarkdownWriter) Write(p []byte) (int, error) {
	if m.raw || !m.isTTY {
		return m.out.Write(p)
	}
	return m.buf.Write(p)
}

// Flush renders the buffered content as styled terminal markdown and writes it
// to the underlying writer. In raw or non-TTY mode this is a no-op.
//
// If glamour renderer initialisation or rendering fails, Flush falls back to
// emitting the raw buffered content and prints a warning to stderr.
func (m *MarkdownWriter) Flush() error {
	if m.raw || !m.isTTY || m.buf.Len() == 0 {
		return nil
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, Muted.Render("  (markdown rendering unavailable, showing raw output)"))
		_, werr := m.out.Write(m.buf.Bytes())
		return werr
	}

	rendered, err := r.Render(m.buf.String())
	if err != nil {
		fmt.Fprintln(os.Stderr, Muted.Render("  (markdown rendering failed, showing raw output)"))
		_, werr := m.out.Write(m.buf.Bytes())
		return werr
	}

	_, err = fmt.Fprint(m.out, rendered)
	return err
}

// RenderMarkdown renders a complete markdown string for terminal output and
// returns the styled result. Returns the original string on any error.
func RenderMarkdown(md string) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return out
}
