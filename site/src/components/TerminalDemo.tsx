import { useState, useEffect, useRef, useCallback } from "react";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type LineType = "command" | "output" | "success" | "muted" | "accent" | "blank";

interface ScriptLine {
  text: string;
  delay: number;
  type: LineType;
}

interface DemoScript {
  label: string;
  lines: ScriptLine[];
}

// ---------------------------------------------------------------------------
// Color palette
// ---------------------------------------------------------------------------

const colors = {
  bg: "#0d0d0d",
  border: "#252525",
  chrome: "#1a1a1a",
  prompt: "#D4A017",
  command: "#e0e0e0",
  output: "#999999",
  success: "#28c840",
  accent: "#D4A017",
  muted: "#666666",
  tabActive: "#D4A017",
  tabInactive: "#666666",
  dotRed: "#ff5f57",
  dotYellow: "#febc2e",
  dotGreen: "#28c840",
} as const;

// ---------------------------------------------------------------------------
// Demo scripts
// ---------------------------------------------------------------------------

const demos: DemoScript[] = [
  {
    label: "Tasks",
    lines: [
      { text: 'mine todo add "ship v0.2" -p high -d tomorrow', delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u2713 Added #4 \u2014 ship v0.2 (high, due tomorrow)", delay: 400, type: "success" },
      { text: "", delay: 600, type: "blank" },
      { text: "mine todo", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u26CF Tasks (3 open)", delay: 300, type: "accent" },
      { text: "", delay: 100, type: "blank" },
      { text: "  #  Pri   Task                Due", delay: 100, type: "muted" },
      { text: "  1  crit  fix auth bug        today", delay: 150, type: "output" },
      { text: "  4  high  ship v0.2           tomorrow", delay: 150, type: "output" },
      { text: "  3  med   update docs         next week", delay: 150, type: "output" },
      { text: "", delay: 600, type: "blank" },
      { text: "mine dig 25m", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u26CF Focus \u2014 25:00", delay: 300, type: "accent" },
      { text: "  \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2591\u2591\u2591\u2591\u2591\u2591\u2591\u2591  12:30 remaining", delay: 300, type: "output" },
      { text: "  \uD83D\uDD25 Streak: 4 days", delay: 300, type: "success" },
    ],
  },
  {
    label: "Git",
    lines: [
      { text: "mine git", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u26CF Select branch", delay: 300, type: "accent" },
      { text: "  > feat/landing-page", delay: 200, type: "success" },
      { text: "    fix/auth-bug", delay: 150, type: "output" },
      { text: "    main", delay: 150, type: "output" },
      { text: "    chore/deps", delay: 150, type: "output" },
      { text: "", delay: 400, type: "blank" },
      { text: "  \u2713 Switched to feat/landing-page", delay: 400, type: "success" },
      { text: "", delay: 600, type: "blank" },
      { text: "mine git changelog", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  ## v0.3.0", delay: 200, type: "accent" },
      { text: "", delay: 100, type: "blank" },
      { text: "  ### Features", delay: 200, type: "output" },
      { text: "  - Add project context switching (#42)", delay: 150, type: "output" },
      { text: "  - AI code review with styled output (#38)", delay: 150, type: "output" },
      { text: "", delay: 100, type: "blank" },
      { text: "  ### Fixes", delay: 200, type: "output" },
      { text: "  - Fix SSH tunnel timeout (#41)", delay: 150, type: "output" },
    ],
  },
  {
    label: "AI",
    lines: [
      { text: "git add . && mine ai review", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u26CF Reviewing staged changes with Claude...", delay: 800, type: "accent" },
      { text: "", delay: 100, type: "blank" },
      { text: "  ## Code Review", delay: 300, type: "output" },
      { text: "", delay: 100, type: "blank" },
      { text: "  **Overall**: Clean implementation. Two suggestions:", delay: 200, type: "output" },
      { text: "", delay: 100, type: "blank" },
      { text: "  1. `internal/proj/proj.go:47` \u2014 Add context.Context", delay: 200, type: "output" },
      { text: "     to Scan() for cancellation support", delay: 150, type: "output" },
      { text: "  2. `cmd/proj.go:89` \u2014 Consider extracting picker", delay: 200, type: "output" },
      { text: "     logic into a shared helper", delay: 150, type: "output" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u2713 Review complete (1.2s)", delay: 400, type: "success" },
      { text: "", delay: 600, type: "blank" },
      { text: "mine ai commit", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u26CF Generating commit message...", delay: 600, type: "accent" },
      { text: "", delay: 100, type: "blank" },
      { text: "  feat: add project context switching", delay: 300, type: "output" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u2713 Committed with generated message", delay: 400, type: "success" },
    ],
  },
  {
    label: "Projects",
    lines: [
      { text: "mine proj scan ~/dev", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u2713 Added 12 projects", delay: 400, type: "success" },
      { text: "  \u25CF ~/dev/mine", delay: 150, type: "output" },
      { text: "  \u25CF ~/dev/api-server", delay: 150, type: "output" },
      { text: "  \u25CF ~/dev/dotfiles", delay: 150, type: "output" },
      { text: "", delay: 600, type: "blank" },
      { text: "mine proj", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u26CF Select project", delay: 300, type: "accent" },
      { text: "  > mine               ~/dev/mine", delay: 200, type: "success" },
      { text: "    api-server          ~/dev/api-server", delay: 150, type: "output" },
      { text: "    dotfiles            ~/dev/dotfiles", delay: 150, type: "output" },
      { text: "", delay: 400, type: "blank" },
      { text: "  \u2713 Selected mine", delay: 400, type: "success" },
      { text: "", delay: 600, type: "blank" },
      { text: "p api-server", delay: 600, type: "command" },
      { text: "", delay: 100, type: "blank" },
      { text: "  \u2713 Switched to api-server (~/dev/api-server)", delay: 400, type: "success" },
    ],
  },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Returns the inline style color for a given line type. */
function lineColor(type: LineType): string {
  switch (type) {
    case "command":
      return colors.command;
    case "output":
      return colors.output;
    case "success":
      return colors.success;
    case "accent":
      return colors.accent;
    case "muted":
      return colors.muted;
    case "blank":
      return "transparent";
  }
}

// ---------------------------------------------------------------------------
// Rendered line types
// ---------------------------------------------------------------------------

interface RenderedLine {
  id: number;
  type: LineType;
  /** For commands, this grows character-by-character; for others, full text. */
  text: string;
  /** Whether this command line is still being typed. */
  typing: boolean;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

const CHAR_DELAY = 35; // ms per character for command typing
const RESTART_PAUSE = 5000; // ms before cycling to next demo
const INACTIVITY_RESUME = 10000; // ms after manual tab click before auto-cycle resumes

export default function TerminalDemo() {
  const [activeDemo, setActiveDemo] = useState(0);
  const [lines, setLines] = useState<RenderedLine[]>([]);
  const [playing, setPlaying] = useState(true);
  const [cursorVisible, setCursorVisible] = useState(true);

  const scrollRef = useRef<HTMLDivElement>(null);
  const cancelRef = useRef<(() => void) | null>(null);
  const lineIdRef = useRef(0);
  const pausedRef = useRef(false);
  const inactivityTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Auto-scroll to bottom whenever lines change
  useEffect(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines]);

  // Cursor blink
  useEffect(() => {
    const id = setInterval(() => setCursorVisible((v) => !v), 530);
    return () => clearInterval(id);
  }, []);

  // ------------------------------------------------------------------
  // Animation runner
  // ------------------------------------------------------------------

  const runDemo = useCallback(
    (demoIndex: number) => {
      // Cancel any in-flight animation
      if (cancelRef.current) {
        cancelRef.current();
      }

      let cancelled = false;
      cancelRef.current = () => {
        cancelled = true;
      };

      setLines([]);
      setActiveDemo(demoIndex);
      setPlaying(true);

      const script = demos[demoIndex].lines;

      // We build up lines incrementally using timeouts.
      const pending: ReturnType<typeof setTimeout>[] = [];

      const schedule = (fn: () => void, ms: number) => {
        const id = setTimeout(() => {
          if (!cancelled) fn();
        }, ms);
        pending.push(id);
      };

      // Walk through script lines, accumulating delay.
      let cumulativeDelay = 300; // initial pause before first line

      for (let i = 0; i < script.length; i++) {
        const line = script[i];

        if (line.type === "command") {
          // Phase 1: show empty command line (prompt appears, typing starts)
          const lineId = ++lineIdRef.current;
          const startDelay = cumulativeDelay;

          schedule(() => {
            setLines((prev) => [
              ...prev,
              { id: lineId, type: "command", text: "", typing: true },
            ]);
          }, startDelay);

          // Phase 2: type each character
          for (let c = 0; c < line.text.length; c++) {
            const charDelay = startDelay + (c + 1) * CHAR_DELAY;
            const partial = line.text.slice(0, c + 1);
            schedule(() => {
              setLines((prev) =>
                prev.map((l) =>
                  l.id === lineId ? { ...l, text: partial } : l,
                ),
              );
            }, charDelay);
          }

          // Phase 3: mark done typing
          const finishTyping = startDelay + line.text.length * CHAR_DELAY;
          schedule(() => {
            setLines((prev) =>
              prev.map((l) =>
                l.id === lineId ? { ...l, typing: false } : l,
              ),
            );
          }, finishTyping);

          cumulativeDelay = finishTyping + line.delay;
        } else {
          // Non-command lines appear instantly after their delay
          const lineId = ++lineIdRef.current;
          const appearAt = cumulativeDelay + line.delay;

          schedule(() => {
            setLines((prev) => [
              ...prev,
              { id: lineId, type: line.type, text: line.text, typing: false },
            ]);
          }, appearAt);

          cumulativeDelay = appearAt;
        }
      }

      // After script finishes, wait and then advance (if not paused)
      schedule(() => {
        setPlaying(false);
        if (!pausedRef.current) {
          schedule(() => {
            if (!cancelled && !pausedRef.current) {
              const next = (demoIndex + 1) % demos.length;
              runDemo(next);
            }
          }, RESTART_PAUSE);
        }
      }, cumulativeDelay + 400);

      // Cleanup: cancel all pending timeouts
      cancelRef.current = () => {
        cancelled = true;
        pending.forEach(clearTimeout);
      };
    },
    [], // no deps -- uses refs for mutable state
  );

  // Start on mount
  useEffect(() => {
    runDemo(0);
    return () => {
      if (cancelRef.current) cancelRef.current();
    };
  }, [runDemo]);

  // ------------------------------------------------------------------
  // Tab click handler
  // ------------------------------------------------------------------

  const handleTabClick = (index: number) => {
    // Pause auto-cycle
    pausedRef.current = true;
    if (inactivityTimerRef.current) {
      clearTimeout(inactivityTimerRef.current);
    }

    // Jump to selected demo
    runDemo(index);

    // Resume auto-cycle after inactivity
    inactivityTimerRef.current = setTimeout(() => {
      pausedRef.current = false;
    }, INACTIVITY_RESUME);
  };

  // Cleanup inactivity timer on unmount
  useEffect(() => {
    return () => {
      if (inactivityTimerRef.current) clearTimeout(inactivityTimerRef.current);
    };
  }, []);

  // ------------------------------------------------------------------
  // Render
  // ------------------------------------------------------------------

  return (
    <div
      style={{
        fontFamily:
          'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace',
        maxWidth: 680,
        margin: "0 auto",
        borderRadius: 12,
        overflow: "hidden",
        border: `1px solid ${colors.border}`,
        background: colors.bg,
        boxShadow:
          "0 25px 60px rgba(0,0,0,0.5), 0 0 80px rgba(212,160,23,0.03)",
      }}
    >
      {/* Title bar */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          background: colors.chrome,
          padding: "12px 16px",
        }}
      >
        <div
          style={{
            width: 12,
            height: 12,
            borderRadius: "50%",
            background: colors.dotRed,
          }}
        />
        <div
          style={{
            width: 12,
            height: 12,
            borderRadius: "50%",
            background: colors.dotYellow,
          }}
        />
        <div
          style={{
            width: 12,
            height: 12,
            borderRadius: "50%",
            background: colors.dotGreen,
          }}
        />
        <div
          style={{
            flex: 1,
            textAlign: "center",
            fontSize: 12,
            color: colors.muted,
            marginRight: 36, // offset for dots
            userSelect: "none",
          }}
        >
          ~ mine
        </div>
      </div>

      {/* Tabs */}
      <div
        style={{
          display: "flex",
          gap: 0,
          background: colors.chrome,
          borderBottom: `1px solid ${colors.border}`,
          paddingLeft: 16,
          paddingRight: 16,
        }}
      >
        {demos.map((demo, i) => (
          <button
            key={demo.label}
            onClick={() => handleTabClick(i)}
            style={{
              background: "none",
              border: "none",
              borderBottom:
                i === activeDemo
                  ? `2px solid ${colors.tabActive}`
                  : "2px solid transparent",
              color: i === activeDemo ? colors.tabActive : colors.tabInactive,
              fontSize: 12,
              fontFamily: "inherit",
              padding: "8px 14px",
              cursor: "pointer",
              transition: "color 0.2s, border-color 0.2s",
              fontWeight: i === activeDemo ? 600 : 400,
              letterSpacing: 0.3,
            }}
          >
            {demo.label}
          </button>
        ))}
      </div>

      {/* Terminal content */}
      <div
        ref={scrollRef}
        style={{
          height: 400,
          overflowY: "auto",
          padding: "20px 24px",
          fontSize: "0.84rem",
          lineHeight: 1.85,
          scrollbarWidth: "thin",
          scrollbarColor: `${colors.border} transparent`,
        }}
      >
        {lines.map((line) => {
          if (line.type === "blank") {
            return (
              <div key={line.id} style={{ height: "1.85em" }} />
            );
          }

          if (line.type === "command") {
            return (
              <div key={line.id} style={{ whiteSpace: "pre" }}>
                <span style={{ color: colors.prompt, fontWeight: 600 }}>
                  ${" "}
                </span>
                <span style={{ color: colors.command }}>{line.text}</span>
                {line.typing && (
                  <span
                    style={{
                      display: "inline-block",
                      width: "0.55em",
                      height: "1.1em",
                      background: cursorVisible ? colors.command : "transparent",
                      marginLeft: 1,
                      verticalAlign: "text-bottom",
                      transition: "background 0.08s",
                    }}
                  />
                )}
              </div>
            );
          }

          return (
            <div
              key={line.id}
              style={{ color: lineColor(line.type), whiteSpace: "pre" }}
            >
              {line.text}
            </div>
          );
        })}

        {/* Resting cursor after script finishes */}
        {!playing && lines.length > 0 && (
          <div style={{ whiteSpace: "pre" }}>
            <span style={{ color: colors.prompt, fontWeight: 600 }}>
              ${" "}
            </span>
            <span
              style={{
                display: "inline-block",
                width: "0.55em",
                height: "1.1em",
                background: cursorVisible ? colors.command : "transparent",
                verticalAlign: "text-bottom",
                transition: "background 0.08s",
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
}
