import { useState, useCallback, useRef, useEffect, type CSSProperties, type KeyboardEvent } from "react";

// ---------------------------------------------------------------------------
// Data
// ---------------------------------------------------------------------------

interface Feature {
  id: string;
  icon: string;
  label: string;
  title: string;
  description: string;
  /** Each line is either a command string or { comment: string } */
  lines: Array<string | { comment: string }>;
}

const FEATURES: Feature[] = [
  {
    id: "tasks",
    icon: "\u{1F4CB}",
    label: "Tasks",
    title: "Task Management",
    description:
      "Fast local task management with priorities, due dates, and tags. Interactive fuzzy-search picker lets you browse and complete tasks without leaving the terminal.",
    lines: [
      'mine todo add "ship v0.2" -p high -d tomorrow',
      { comment: "# interactive picker" },
      "mine todo",
    ],
  },
  {
    id: "focus",
    icon: "\u{1F525}",
    label: "Focus",
    title: "Focus Timer",
    description:
      "Pomodoro-style deep work timer with streak tracking. Full-screen countdown keeps you locked in. Stats track your deep work hours over time.",
    lines: [
      { comment: "# start a 25-minute focus session" },
      "mine dig 25m",
      "mine dig stats",
    ],
  },
  {
    id: "git",
    icon: "\u{1F500}",
    label: "Git",
    title: "Git Workflow",
    description:
      "Fuzzy branch picker, stale branch sweep, WIP commits, PR creation, and changelog generation from conventional commits.",
    lines: [
      { comment: "# fuzzy branch picker" },
      "mine git",
      "mine git changelog",
      "mine git pr",
    ],
  },
  {
    id: "scaffold",
    icon: "\u{1F4E6}",
    label: "Scaffold",
    title: "Project Scaffolding",
    description:
      "Bootstrap Go, Node, Python, or Rust projects in one command. Built-in recipes or drop your own templates into the config directory.",
    lines: ["mine craft dev go", "mine craft dev node", "mine craft list"],
  },
  {
    id: "ssh",
    icon: "\u{1F4BB}",
    label: "SSH",
    title: "SSH Management",
    description:
      "Fuzzy host picker from your SSH config, ed25519 key generation, tunnel shortcuts, and key distribution.",
    lines: [
      { comment: "# fuzzy host picker" },
      "mine ssh",
      "mine ssh tunnel myhost 8080",
      "mine ssh keygen",
    ],
  },
  {
    id: "vault",
    icon: "\u{1F512}",
    label: "Vault",
    title: "Secrets Vault",
    description:
      "Age-encrypted secret storage. Set and get secrets with dot-notation keys. Clipboard integration keeps secrets out of terminal history.",
    lines: [
      "mine vault set api.openai sk-...",
      "mine vault get api.openai --clip",
    ],
  },
  {
    id: "ai",
    icon: "\u{2728}",
    label: "AI",
    title: "AI Integration",
    description:
      "Multi-provider AI for code review and commit messages. Supports Claude, OpenAI, Gemini. Styled markdown output in the terminal.",
    lines: [
      "mine ai review",
      "mine ai commit",
      'mine ai ask "explain goroutines"',
    ],
  },
  {
    id: "projects",
    icon: "\u{1F4C1}",
    label: "Projects",
    title: "Project Context",
    description:
      "Register projects, scan directories for git repos, and switch contexts instantly. Per-project settings and environment profiles.",
    lines: [
      "mine proj scan ~/dev",
      { comment: "# project picker" },
      "mine proj",
      "p api-server",
    ],
  },
  {
    id: "tmux",
    icon: "\u{1FA9F}",
    label: "Tmux",
    title: "Tmux Sessions",
    description:
      "Fuzzy session picker, layout save and restore, auto-naming from directory. Never lose your tmux setup again.",
    lines: [
      { comment: "# session picker" },
      "mine tmux",
      "mine tmux save",
      "mine tmux load dev",
    ],
  },
];

// ---------------------------------------------------------------------------
// Palette (inline styles, zero CSS deps)
// ---------------------------------------------------------------------------

const C = {
  gold: "#D4A017",
  goldBg: "rgba(212, 160, 23, 0.08)",
  inactiveText: "#999999",
  hoverBg: "rgba(255, 255, 255, 0.03)",
  codeBg: "#0d0d0d",
  codeBorder: "#252525",
  codeText: "#e0e0e0",
  codeComment: "#666666",
  descText: "#cccccc",
  titleText: "#e0e0e0",
} as const;

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function CodeBlock({ lines }: { lines: Feature["lines"] }) {
  const style: CSSProperties = {
    background: C.codeBg,
    border: `1px solid ${C.codeBorder}`,
    borderRadius: 8,
    padding: "20px 24px",
    fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', Menlo, Consolas, monospace",
    fontSize: 14,
    lineHeight: 1.7,
    overflowX: "auto",
    margin: 0,
  };

  return (
    <pre style={style}>
      <code>
        {lines.map((line, i) => {
          if (typeof line === "object" && "comment" in line) {
            return (
              <span key={i} style={{ color: C.codeComment }}>
                {line.comment}
                {"\n"}
              </span>
            );
          }
          return (
            <span key={i}>
              <span style={{ color: C.gold }}>$ </span>
              <span style={{ color: C.codeText }}>{line}</span>
              {"\n"}
            </span>
          );
        })}
      </code>
    </pre>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export default function FeatureTabs() {
  const [activeIndex, setActiveIndex] = useState(0);
  const [fadeKey, setFadeKey] = useState(0);
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);

  // Trigger a fresh fade-in whenever the tab changes
  const selectTab = useCallback(
    (index: number) => {
      if (index === activeIndex) return;
      setActiveIndex(index);
      setFadeKey((k) => k + 1);
    },
    [activeIndex],
  );

  // Keep ref array sized correctly
  useEffect(() => {
    tabRefs.current = tabRefs.current.slice(0, FEATURES.length);
  }, []);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLDivElement>, isMobile: boolean) => {
      const prev = isMobile ? "ArrowLeft" : "ArrowUp";
      const next = isMobile ? "ArrowRight" : "ArrowDown";

      let newIndex = activeIndex;
      if (e.key === prev) {
        e.preventDefault();
        newIndex = (activeIndex - 1 + FEATURES.length) % FEATURES.length;
      } else if (e.key === next) {
        e.preventDefault();
        newIndex = (activeIndex + 1) % FEATURES.length;
      } else if (e.key === "Home") {
        e.preventDefault();
        newIndex = 0;
      } else if (e.key === "End") {
        e.preventDefault();
        newIndex = FEATURES.length - 1;
      } else {
        return;
      }

      selectTab(newIndex);
      tabRefs.current[newIndex]?.focus();
    },
    [activeIndex, selectTab],
  );

  const active = FEATURES[activeIndex];

  // ---- Styles ----

  const container: CSSProperties = {
    maxWidth: 1100,
    margin: "0 auto",
    width: "100%",
  };

  const desktopLayout: CSSProperties = {
    display: "flex",
    gap: 32,
  };

  const tabListDesktop: CSSProperties = {
    display: "flex",
    flexDirection: "column",
    gap: 2,
    minWidth: 200,
    width: "30%",
    flexShrink: 0,
  };

  const tabListMobile: CSSProperties = {
    display: "flex",
    gap: 4,
    overflowX: "auto",
    paddingBottom: 8,
    WebkitOverflowScrolling: "touch",
    scrollbarWidth: "none",
    msOverflowStyle: "none",
  };

  const panelDesktop: CSSProperties = {
    flex: 1,
    minWidth: 0,
  };

  const tabBtnBase: CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: 10,
    padding: "12px 16px",
    border: "none",
    background: "transparent",
    cursor: "pointer",
    fontSize: 15,
    fontFamily: "inherit",
    textAlign: "left",
    borderRadius: 6,
    transition: "background 0.15s, color 0.15s, border-color 0.15s",
    whiteSpace: "nowrap",
    outline: "none",
  };

  const activeTabDesktop: CSSProperties = {
    ...tabBtnBase,
    color: C.gold,
    background: C.goldBg,
    borderLeft: `3px solid ${C.gold}`,
    paddingLeft: 13,
  };

  const inactiveTabDesktop: CSSProperties = {
    ...tabBtnBase,
    color: C.inactiveText,
    borderLeft: "3px solid transparent",
    paddingLeft: 13,
  };

  const activeTabMobile: CSSProperties = {
    ...tabBtnBase,
    color: C.gold,
    background: C.goldBg,
    borderBottom: `2px solid ${C.gold}`,
    borderRadius: "6px 6px 0 0",
    padding: "10px 16px",
  };

  const inactiveTabMobile: CSSProperties = {
    ...tabBtnBase,
    color: C.inactiveText,
    borderBottom: "2px solid transparent",
    borderRadius: "6px 6px 0 0",
    padding: "10px 16px",
  };

  // Fade-in keyframes via inline style (no CSS file needed)
  const panelContent: CSSProperties = {
    animation: "featuretabs-fadein 0.25s ease-out",
  };

  // ---- Shared tab rendering ----

  const renderTab = (
    feature: Feature,
    index: number,
    isActive: boolean,
    variant: "desktop" | "mobile",
  ) => {
    const style =
      variant === "desktop"
        ? isActive
          ? activeTabDesktop
          : inactiveTabDesktop
        : isActive
          ? activeTabMobile
          : inactiveTabMobile;

    return (
      <button
        key={feature.id}
        ref={(el) => {
          tabRefs.current[index] = el;
        }}
        role="tab"
        id={`featuretab-${feature.id}`}
        aria-selected={isActive}
        aria-controls="featuretab-panel"
        tabIndex={isActive ? 0 : -1}
        style={style}
        onClick={() => selectTab(index)}
        onMouseEnter={(e) => {
          if (!isActive) {
            (e.currentTarget as HTMLButtonElement).style.background = C.hoverBg;
          }
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLButtonElement).style.background = isActive
            ? C.goldBg
            : "transparent";
        }}
        onFocus={(e) => {
          if (!isActive) {
            (e.currentTarget as HTMLButtonElement).style.background = C.hoverBg;
          }
        }}
        onBlur={(e) => {
          (e.currentTarget as HTMLButtonElement).style.background = isActive
            ? C.goldBg
            : "transparent";
        }}
      >
        <span style={{ fontSize: 18, lineHeight: 1 }} aria-hidden="true">
          {feature.icon}
        </span>
        <span>{feature.label}</span>
      </button>
    );
  };

  // ---- Panel ----

  const panel = (
    <div
      role="tabpanel"
      id="featuretab-panel"
      aria-labelledby={`featuretab-${active.id}`}
      key={fadeKey}
      style={panelContent}
    >
      <h3
        style={{
          margin: "0 0 12px",
          fontSize: 24,
          fontWeight: 600,
          color: C.titleText,
        }}
      >
        {active.icon} {active.title}
      </h3>
      <p
        style={{
          margin: "0 0 20px",
          fontSize: 15,
          lineHeight: 1.65,
          color: C.descText,
        }}
      >
        {active.description}
      </p>
      <CodeBlock lines={active.lines} />
    </div>
  );

  return (
    <>
      {/* Inject keyframe animation once */}
      <style>{`
        @keyframes featuretabs-fadein {
          from { opacity: 0; transform: translateY(6px); }
          to   { opacity: 1; transform: translateY(0); }
        }
        /* Hide scrollbar for mobile tab list */
        .featuretabs-mobile-tablist::-webkit-scrollbar { display: none; }
      `}</style>

      <div style={container}>
        {/* Desktop layout (>768px) */}
        <div
          style={{ ...desktopLayout }}
          className="featuretabs-desktop"
        >
          <div
            role="tablist"
            aria-label="Features"
            aria-orientation="vertical"
            style={tabListDesktop}
            onKeyDown={(e) => handleKeyDown(e, false)}
          >
            {FEATURES.map((f, i) =>
              renderTab(f, i, i === activeIndex, "desktop"),
            )}
          </div>
          <div style={panelDesktop}>{panel}</div>
        </div>

        {/* Mobile layout (<=768px) */}
        <div className="featuretabs-mobile">
          <div
            role="tablist"
            aria-label="Features"
            aria-orientation="horizontal"
            style={tabListMobile}
            className="featuretabs-mobile-tablist"
            onKeyDown={(e) => handleKeyDown(e, true)}
          >
            {FEATURES.map((f, i) =>
              renderTab(f, i, i === activeIndex, "mobile"),
            )}
          </div>
          <div style={{ marginTop: 16 }}>{panel}</div>
        </div>
      </div>

      {/* Responsive visibility via inline <style> */}
      <style>{`
        .featuretabs-mobile { display: none; }
        .featuretabs-desktop { display: flex; }
        @media (max-width: 768px) {
          .featuretabs-mobile { display: block; }
          .featuretabs-desktop { display: none !important; }
        }
      `}</style>
    </>
  );
}
