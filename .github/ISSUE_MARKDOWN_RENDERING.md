# Add terminal markdown rendering for AI output

## Overview

AI responses currently output raw markdown text which doesn't render properly in the terminal. This makes code blocks, headings, lists, and other formatted content hard to read.

## Proposed Solution

Use the `glamour` library from charmbracelet (same ecosystem as lipgloss/bubbletea) to render markdown beautifully in the terminal.

**Library**: https://github.com/charmbracelet/glamour

## Implementation Notes

- Apply markdown rendering to AI streaming and complete responses
- Use a terminal-friendly style (dark mode support)
- Maintain current streaming UX (render as content arrives)
- Consider adding a `--raw` flag to disable rendering for piping

## Example Usage

```go
import "github.com/charmbracelet/glamour"

renderer, _ := glamour.NewTermRenderer(
    glamour.WithAutoStyle(),
    glamour.WithWordWrap(80),
)
out, _ := renderer.Render(markdownContent)
fmt.Print(out)
```

## Acceptance Criteria

- [ ] AI responses render markdown formatting correctly
- [ ] Code blocks are syntax-highlighted
- [ ] Headers, lists, and emphasis are properly styled
- [ ] Works with streaming output
- [ ] Optional raw mode for piping/scripting

## Priority

Medium - Improves UX significantly but not blocking functionality

## Labels

enhancement, phase/2
