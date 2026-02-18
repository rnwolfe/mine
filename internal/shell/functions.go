package shell

import "fmt"

// ShellFunc describes a shell utility function.
type ShellFunc struct {
	Name string
	Desc string
	Bash string
	Zsh  string
	Fish string
}

// Functions returns the built-in shell function library.
func Functions() []ShellFunc {
	return []ShellFunc{
		{
			Name: "mkcd",
			Desc: "Create a directory and cd into it",
			Bash: `mkcd() {
  if [ "$1" = "--help" ]; then
    echo "mkcd — Create a directory and cd into it"
    echo "Usage: mkcd <dir>"
    echo "Example: mkcd my-project"
    return 0
  fi
  mkdir -p "$1" && cd "$1"
}`,
			Zsh: `mkcd() {
  if [[ "$1" == "--help" ]]; then
    echo "mkcd — Create a directory and cd into it"
    echo "Usage: mkcd <dir>"
    echo "Example: mkcd my-project"
    return 0
  fi
  mkdir -p "$1" && cd "$1"
}`,
			Fish: `function mkcd
  if test "$argv[1]" = "--help"
    echo "mkcd — Create a directory and cd into it"
    echo "Usage: mkcd <dir>"
    echo "Example: mkcd my-project"
    return 0
  end
  mkdir -p $argv[1]; and cd $argv[1]
end`,
		},
		{
			Name: "extract",
			Desc: "Extract any common archive format",
			Bash: `extract() {
  if [ "$1" = "--help" ]; then
    echo "extract — Extract any common archive format"
    echo "Usage: extract <file>"
    echo "Example: extract archive.tar.gz"
    return 0
  fi
  if [ ! -f "$1" ]; then echo "extract: '$1' not found" >&2; return 1; fi
  case "$1" in
    *.tar.bz2) tar xjf "$1" ;;
    *.tar.gz)  tar xzf "$1" ;;
    *.tar.xz)  tar xJf "$1" ;;
    *.tar.zst) tar --zstd -xf "$1" ;;
    *.bz2)     bunzip2 "$1" ;;
    *.gz)      gunzip "$1" ;;
    *.tar)     tar xf "$1" ;;
    *.tbz2)    tar xjf "$1" ;;
    *.tgz)     tar xzf "$1" ;;
    *.zip)     unzip "$1" ;;
    *.Z)       uncompress "$1" ;;
    *.7z)      7z x "$1" ;;
    *)         echo "extract: unknown format '$1'" >&2; return 1 ;;
  esac
}`,
			Zsh: `extract() {
  if [[ "$1" == "--help" ]]; then
    echo "extract — Extract any common archive format"
    echo "Usage: extract <file>"
    echo "Example: extract archive.tar.gz"
    return 0
  fi
  if [[ ! -f "$1" ]]; then echo "extract: '$1' not found" >&2; return 1; fi
  case "$1" in
    *.tar.bz2) tar xjf "$1" ;;
    *.tar.gz)  tar xzf "$1" ;;
    *.tar.xz)  tar xJf "$1" ;;
    *.tar.zst) tar --zstd -xf "$1" ;;
    *.bz2)     bunzip2 "$1" ;;
    *.gz)      gunzip "$1" ;;
    *.tar)     tar xf "$1" ;;
    *.tbz2)    tar xjf "$1" ;;
    *.tgz)     tar xzf "$1" ;;
    *.zip)     unzip "$1" ;;
    *.Z)       uncompress "$1" ;;
    *.7z)      7z x "$1" ;;
    *)         echo "extract: unknown format '$1'" >&2; return 1 ;;
  esac
}`,
			Fish: `function extract
  if test "$argv[1]" = "--help"
    echo "extract — Extract any common archive format"
    echo "Usage: extract <file>"
    echo "Example: extract archive.tar.gz"
    return 0
  end
  if not test -f $argv[1]
    echo "extract: '$argv[1]' not found" >&2; return 1
  end
  switch $argv[1]
    case '*.tar.bz2'; tar xjf $argv[1]
    case '*.tar.gz';  tar xzf $argv[1]
    case '*.tar.xz';  tar xJf $argv[1]
    case '*.tar.zst'; tar --zstd -xf $argv[1]
    case '*.bz2';     bunzip2 $argv[1]
    case '*.gz';      gunzip $argv[1]
    case '*.tar';     tar xf $argv[1]
    case '*.tbz2';    tar xjf $argv[1]
    case '*.tgz';     tar xzf $argv[1]
    case '*.zip';     unzip $argv[1]
    case '*.Z';       uncompress $argv[1]
    case '*.7z';      7z x $argv[1]
    case '*';         echo "extract: unknown format '$argv[1]'" >&2; return 1
  end
end`,
		},
		{
			Name: "ports",
			Desc: "Show listening network ports",
			Bash: `ports() {
  if [ "$1" = "--help" ]; then
    echo "ports — Show listening network ports"
    echo "Usage: ports"
    echo "Example: ports"
    return 0
  fi
  lsof -iTCP -sTCP:LISTEN -P -n 2>/dev/null || ss -tlnp 2>/dev/null
}`,
			Zsh: `ports() {
  if [[ "$1" == "--help" ]]; then
    echo "ports — Show listening network ports"
    echo "Usage: ports"
    echo "Example: ports"
    return 0
  fi
  lsof -iTCP -sTCP:LISTEN -P -n 2>/dev/null || ss -tlnp 2>/dev/null
}`,
			Fish: `function ports
  if test "$argv[1]" = "--help"
    echo "ports — Show listening network ports"
    echo "Usage: ports"
    echo "Example: ports"
    return 0
  end
  lsof -iTCP -sTCP:LISTEN -P -n 2>/dev/null; or ss -tlnp 2>/dev/null
end`,
		},
		{
			Name: "gitroot",
			Desc: "cd to the root of the current git repo",
			Bash: `gitroot() {
  if [ "$1" = "--help" ]; then
    echo "gitroot — cd to the root of the current git repo"
    echo "Usage: gitroot"
    echo "Example: gitroot"
    return 0
  fi
  cd "$(git rev-parse --show-toplevel 2>/dev/null)" || echo "not in a git repo" >&2
}`,
			Zsh: `gitroot() {
  if [[ "$1" == "--help" ]]; then
    echo "gitroot — cd to the root of the current git repo"
    echo "Usage: gitroot"
    echo "Example: gitroot"
    return 0
  fi
  cd "$(git rev-parse --show-toplevel 2>/dev/null)" || echo "not in a git repo" >&2
}`,
			Fish: `function gitroot
  if test "$argv[1]" = "--help"
    echo "gitroot — cd to the root of the current git repo"
    echo "Usage: gitroot"
    echo "Example: gitroot"
    return 0
  end
  cd (git rev-parse --show-toplevel 2>/dev/null); or echo "not in a git repo" >&2
end`,
		},
		{
			Name: "serve",
			Desc: "Start a quick HTTP server in the current directory",
			Bash: `serve() {
  if [ "$1" = "--help" ]; then
    echo "serve — Start a quick HTTP server in the current directory"
    echo "Usage: serve [port]"
    echo "Example: serve 3000"
    return 0
  fi
  local port="${1:-8000}"
  python3 -m http.server "$port" 2>/dev/null || python -m SimpleHTTPServer "$port"
}`,
			Zsh: `serve() {
  if [[ "$1" == "--help" ]]; then
    echo "serve — Start a quick HTTP server in the current directory"
    echo "Usage: serve [port]"
    echo "Example: serve 3000"
    return 0
  fi
  local port="${1:-8000}"
  python3 -m http.server "$port" 2>/dev/null || python -m SimpleHTTPServer "$port"
}`,
			Fish: `function serve
  if test "$argv[1]" = "--help"
    echo "serve — Start a quick HTTP server in the current directory"
    echo "Usage: serve [port]"
    echo "Example: serve 3000"
    return 0
  end
  set -l port (test (count $argv) -gt 0; and echo $argv[1]; or echo 8000)
  python3 -m http.server $port 2>/dev/null; or python -m SimpleHTTPServer $port
end`,
		},
		{
			Name: "backup",
			Desc: "Create a timestamped backup copy of a file",
			Bash: `backup() {
  if [ "$1" = "--help" ]; then
    echo "backup — Create a timestamped backup copy of a file"
    echo "Usage: backup <file>"
    echo "Example: backup config.yaml"
    return 0
  fi
  cp "$1" "$1.bak.$(date +%Y%m%d_%H%M%S)"
}`,
			Zsh: `backup() {
  if [[ "$1" == "--help" ]]; then
    echo "backup — Create a timestamped backup copy of a file"
    echo "Usage: backup <file>"
    echo "Example: backup config.yaml"
    return 0
  fi
  cp "$1" "$1.bak.$(date +%Y%m%d_%H%M%S)"
}`,
			Fish: `function backup
  if test "$argv[1]" = "--help"
    echo "backup — Create a timestamped backup copy of a file"
    echo "Usage: backup <file>"
    echo "Example: backup config.yaml"
    return 0
  end
  cp $argv[1] $argv[1].bak.(date +%Y%m%d_%H%M%S)
end`,
		},
		{
			Name: "tre",
			Desc: "tree with sensible defaults (2 levels, ignore hidden/vendor)",
			Bash: `tre() {
  if [ "$1" = "--help" ]; then
    echo "tre — tree with sensible defaults (2 levels, ignore hidden/vendor)"
    echo "Usage: tre [depth]"
    echo "Example: tre 3"
    return 0
  fi
  tree -L "${1:-2}" -I 'node_modules|vendor|.git|__pycache__|.venv' --dirsfirst
}`,
			Zsh: `tre() {
  if [[ "$1" == "--help" ]]; then
    echo "tre — tree with sensible defaults (2 levels, ignore hidden/vendor)"
    echo "Usage: tre [depth]"
    echo "Example: tre 3"
    return 0
  fi
  tree -L "${1:-2}" -I 'node_modules|vendor|.git|__pycache__|.venv' --dirsfirst
}`,
			Fish: `function tre
  if test "$argv[1]" = "--help"
    echo "tre — tree with sensible defaults (2 levels, ignore hidden/vendor)"
    echo "Usage: tre [depth]"
    echo "Example: tre 3"
    return 0
  end
  tree -L (test (count $argv) -gt 0; and echo $argv[1]; or echo 2) -I 'node_modules|vendor|.git|__pycache__|.venv' --dirsfirst
end`,
		},
		// --- tmux helpers ---
		{
			Name: "tn",
			Desc: "Quick tmux new-session (defaults to dirname)",
			Bash: `tn() {
  if [ "$1" = "--help" ]; then
    echo "tn — Quick tmux new-session (defaults to dirname)"
    echo "Usage: tn [name]"
    echo "Example: tn myproject"
    return 0
  fi
  local name="${1:-$(basename "$PWD")}"
  tmux new-session -d -s "$name" 2>/dev/null && echo "Session '$name' created" || tmux attach-session -t "$name"
}`,
			Zsh: `tn() {
  if [[ "$1" == "--help" ]]; then
    echo "tn — Quick tmux new-session (defaults to dirname)"
    echo "Usage: tn [name]"
    echo "Example: tn myproject"
    return 0
  fi
  local name="${1:-$(basename "$PWD")}"
  tmux new-session -d -s "$name" 2>/dev/null && echo "Session '$name' created" || tmux attach-session -t "$name"
}`,
			Fish: `function tn
  if test "$argv[1]" = "--help"
    echo "tn — Quick tmux new-session (defaults to dirname)"
    echo "Usage: tn [name]"
    echo "Example: tn myproject"
    return 0
  end
  set -l name (test (count $argv) -gt 0; and echo $argv[1]; or basename $PWD)
  tmux new-session -d -s "$name" 2>/dev/null; and echo "Session '$name' created"; or tmux attach-session -t "$name"
end`,
		},
		{
			Name: "ta",
			Desc: "Attach or switch to a tmux session",
			Bash: `ta() {
  if [ "$1" = "--help" ]; then
    echo "ta — Attach or switch to a tmux session"
    echo "Usage: ta [name]"
    echo "Example: ta myproject"
    return 0
  fi
  if [ -n "$TMUX" ]; then
    tmux switch-client -t "$1"
  else
    tmux attach-session -t "$1"
  fi
}`,
			Zsh: `ta() {
  if [[ "$1" == "--help" ]]; then
    echo "ta — Attach or switch to a tmux session"
    echo "Usage: ta [name]"
    echo "Example: ta myproject"
    return 0
  fi
  if [[ -n "$TMUX" ]]; then
    tmux switch-client -t "$1"
  else
    tmux attach-session -t "$1"
  fi
}`,
			Fish: `function ta
  if test "$argv[1]" = "--help"
    echo "ta — Attach or switch to a tmux session"
    echo "Usage: ta [name]"
    echo "Example: ta myproject"
    return 0
  end
  if set -q TMUX
    tmux switch-client -t $argv[1]
  else
    tmux attach-session -t $argv[1]
  end
end`,
		},
		{
			Name: "tls",
			Desc: "List tmux sessions (compact)",
			Bash: `tls() {
  if [ "$1" = "--help" ]; then
    echo "tls — List tmux sessions (compact)"
    echo "Usage: tls"
    echo "Example: tls"
    return 0
  fi
  tmux list-sessions 2>/dev/null || echo "No tmux sessions running."
}`,
			Zsh: `tls() {
  if [[ "$1" == "--help" ]]; then
    echo "tls — List tmux sessions (compact)"
    echo "Usage: tls"
    echo "Example: tls"
    return 0
  fi
  tmux list-sessions 2>/dev/null || echo "No tmux sessions running."
}`,
			Fish: `function tls
  if test "$argv[1]" = "--help"
    echo "tls — List tmux sessions (compact)"
    echo "Usage: tls"
    echo "Example: tls"
    return 0
  end
  tmux list-sessions 2>/dev/null; or echo "No tmux sessions running."
end`,
		},
		{
			Name: "tk",
			Desc: "Kill a tmux session by name",
			Bash: `tk() {
  if [ "$1" = "--help" ]; then
    echo "tk — Kill a tmux session by name"
    echo "Usage: tk <name>"
    echo "Example: tk myproject"
    return 0
  fi
  tmux kill-session -t "$1"
}`,
			Zsh: `tk() {
  if [[ "$1" == "--help" ]]; then
    echo "tk — Kill a tmux session by name"
    echo "Usage: tk <name>"
    echo "Example: tk myproject"
    return 0
  fi
  tmux kill-session -t "$1"
}`,
			Fish: `function tk
  if test "$argv[1]" = "--help"
    echo "tk — Kill a tmux session by name"
    echo "Usage: tk <name>"
    echo "Example: tk myproject"
    return 0
  end
  tmux kill-session -t $argv[1]
end`,
		},
		{
			Name: "tsp",
			Desc: "Split tmux pane horizontally",
			Bash: `tsp() {
  if [ "$1" = "--help" ]; then
    echo "tsp — Split tmux pane horizontally"
    echo "Usage: tsp"
    echo "Example: tsp"
    return 0
  fi
  tmux split-window -h
}`,
			Zsh: `tsp() {
  if [[ "$1" == "--help" ]]; then
    echo "tsp — Split tmux pane horizontally"
    echo "Usage: tsp"
    echo "Example: tsp"
    return 0
  fi
  tmux split-window -h
}`,
			Fish: `function tsp
  if test "$argv[1]" = "--help"
    echo "tsp — Split tmux pane horizontally"
    echo "Usage: tsp"
    echo "Example: tsp"
    return 0
  end
  tmux split-window -h
end`,
		},
		{
			Name: "tsv",
			Desc: "Split tmux pane vertically",
			Bash: `tsv() {
  if [ "$1" = "--help" ]; then
    echo "tsv — Split tmux pane vertically"
    echo "Usage: tsv"
    echo "Example: tsv"
    return 0
  fi
  tmux split-window -v
}`,
			Zsh: `tsv() {
  if [[ "$1" == "--help" ]]; then
    echo "tsv — Split tmux pane vertically"
    echo "Usage: tsv"
    echo "Example: tsv"
    return 0
  fi
  tmux split-window -v
}`,
			Fish: `function tsv
  if test "$argv[1]" = "--help"
    echo "tsv — Split tmux pane vertically"
    echo "Usage: tsv"
    echo "Example: tsv"
    return 0
  end
  tmux split-window -v
end`,
		},
		// --- git helpers ---
		{
			Name: "gc",
			Desc: "git commit -m shorthand",
			Bash: `gc() {
  if [ "$1" = "--help" ]; then
    echo "gc — git commit -m shorthand"
    echo "Usage: gc <message>"
    echo "Example: gc \"feat: add user auth\""
    return 0
  fi
  git commit -m "$*"
}`,
			Zsh: `gc() {
  if [[ "$1" == "--help" ]]; then
    echo "gc — git commit -m shorthand"
    echo "Usage: gc <message>"
    echo "Example: gc \"feat: add user auth\""
    return 0
  fi
  git commit -m "$*"
}`,
			Fish: `function gc
  if test "$argv[1]" = "--help"
    echo "gc — git commit -m shorthand"
    echo "Usage: gc <message>"
    echo "Example: gc \"feat: add user auth\""
    return 0
  end
  git commit -m "$argv"
end`,
		},
		{
			Name: "gca",
			Desc: "git commit --amend -m shorthand",
			Bash: `gca() {
  if [ "$1" = "--help" ]; then
    echo "gca — git commit --amend -m shorthand"
    echo "Usage: gca <message>"
    echo "Example: gca \"fix: corrected typo\""
    return 0
  fi
  git commit --amend -m "$*"
}`,
			Zsh: `gca() {
  if [[ "$1" == "--help" ]]; then
    echo "gca — git commit --amend -m shorthand"
    echo "Usage: gca <message>"
    echo "Example: gca \"fix: corrected typo\""
    return 0
  fi
  git commit --amend -m "$*"
}`,
			Fish: `function gca
  if test "$argv[1]" = "--help"
    echo "gca — git commit --amend -m shorthand"
    echo "Usage: gca <message>"
    echo "Example: gca \"fix: corrected typo\""
    return 0
  end
  git commit --amend -m "$argv"
end`,
		},
		{
			Name: "gp",
			Desc: "git push with upstream tracking",
			Bash: `gp() {
  if [ "$1" = "--help" ]; then
    echo "gp — git push with upstream tracking"
    echo "Usage: gp"
    echo "Example: gp"
    return 0
  fi
  git push --set-upstream origin "$(git rev-parse --abbrev-ref HEAD 2>/dev/null)"
}`,
			Zsh: `gp() {
  if [[ "$1" == "--help" ]]; then
    echo "gp — git push with upstream tracking"
    echo "Usage: gp"
    echo "Example: gp"
    return 0
  fi
  git push --set-upstream origin "$(git rev-parse --abbrev-ref HEAD 2>/dev/null)"
}`,
			Fish: `function gp
  if test "$argv[1]" = "--help"
    echo "gp — git push with upstream tracking"
    echo "Usage: gp"
    echo "Example: gp"
    return 0
  end
  git push --set-upstream origin (git rev-parse --abbrev-ref HEAD 2>/dev/null)
end`,
		},
		{
			Name: "gpl",
			Desc: "git pull --rebase",
			Bash: `gpl() {
  if [ "$1" = "--help" ]; then
    echo "gpl — git pull --rebase"
    echo "Usage: gpl"
    echo "Example: gpl"
    return 0
  fi
  git pull --rebase
}`,
			Zsh: `gpl() {
  if [[ "$1" == "--help" ]]; then
    echo "gpl — git pull --rebase"
    echo "Usage: gpl"
    echo "Example: gpl"
    return 0
  fi
  git pull --rebase
}`,
			Fish: `function gpl
  if test "$argv[1]" = "--help"
    echo "gpl — git pull --rebase"
    echo "Usage: gpl"
    echo "Example: gpl"
    return 0
  end
  git pull --rebase
end`,
		},
		{
			Name: "gsw",
			Desc: "git switch shorthand",
			Bash: `gsw() {
  if [ "$1" = "--help" ]; then
    echo "gsw — git switch shorthand"
    echo "Usage: gsw <branch>"
    echo "Example: gsw main"
    return 0
  fi
  git switch "$1"
}`,
			Zsh: `gsw() {
  if [[ "$1" == "--help" ]]; then
    echo "gsw — git switch shorthand"
    echo "Usage: gsw <branch>"
    echo "Example: gsw main"
    return 0
  fi
  git switch "$1"
}`,
			Fish: `function gsw
  if test "$argv[1]" = "--help"
    echo "gsw — git switch shorthand"
    echo "Usage: gsw <branch>"
    echo "Example: gsw main"
    return 0
  end
  git switch $argv[1]
end`,
		},
		// --- ssh helpers ---
		{
			Name: "sc",
			Desc: "Quick SSH connect (sc <alias>)",
			Bash: `sc() {
  if [ "$1" = "--help" ]; then
    echo "sc — Quick SSH connect"
    echo "Usage: sc <alias>"
    echo "Example: sc myserver"
    return 0
  fi
  if [ -z "$1" ]; then echo "sc: alias required" >&2; return 1; fi
  ssh "$1"
}`,
			Zsh: `sc() {
  if [[ "$1" == "--help" ]]; then
    echo "sc — Quick SSH connect"
    echo "Usage: sc <alias>"
    echo "Example: sc myserver"
    return 0
  fi
  if [[ -z "$1" ]]; then echo "sc: alias required" >&2; return 1; fi
  ssh "$1"
}`,
			Fish: `function sc
  if test "$argv[1]" = "--help"
    echo "sc — Quick SSH connect"
    echo "Usage: sc <alias>"
    echo "Example: sc myserver"
    return 0
  end
  if test (count $argv) -eq 0
    echo "sc: alias required" >&2; return 1
  end
  ssh "$argv[1]"
end`,
		},
		{
			Name: "scp2",
			Desc: "scp wrapper with progress and resume (rsync over SSH)",
			Bash: `scp2() {
  if [ "$1" = "--help" ]; then
    echo "scp2 — scp wrapper with progress and resume"
    echo "Usage: scp2 <src> <dest>"
    echo "Example: scp2 myserver:/remote/file ./local/"
    return 0
  fi
  if [ $# -lt 2 ]; then echo "scp2: usage: scp2 <src> <dest>" >&2; return 1; fi
  rsync -avzP --partial "$1" "$2"
}`,
			Zsh: `scp2() {
  if [[ "$1" == "--help" ]]; then
    echo "scp2 — scp wrapper with progress and resume"
    echo "Usage: scp2 <src> <dest>"
    echo "Example: scp2 myserver:/remote/file ./local/"
    return 0
  fi
  if [[ $# -lt 2 ]]; then echo "scp2: usage: scp2 <src> <dest>" >&2; return 1; fi
  rsync -avzP --partial "$1" "$2"
}`,
			Fish: `function scp2
  if test "$argv[1]" = "--help"
    echo "scp2 — scp wrapper with progress and resume"
    echo "Usage: scp2 <src> <dest>"
    echo "Example: scp2 myserver:/remote/file ./local/"
    return 0
  end
  if test (count $argv) -lt 2
    echo "scp2: usage: scp2 <src> <dest>" >&2; return 1
  end
  rsync -avzP --partial "$argv[1]" "$argv[2]"
end`,
		},
		{
			Name: "stun",
			Desc: "Quick SSH tunnel shorthand (stun <alias> <local:remote>)",
			Bash: `stun() {
  if [ "$1" = "--help" ]; then
    echo "stun — Quick SSH tunnel shorthand"
    echo "Usage: stun <alias> <local:remote> or stun <alias> <local:remotehost:remote>"
    echo "Example: stun myserver 8080:80"
    echo "Example: stun myserver 5433:db.internal:5432"
    return 0
  fi
  if [ $# -lt 2 ]; then echo "stun: usage: stun <alias> <local:remote>" >&2; return 1; fi
  local _alias="$1"
  local _spec="$2"
  local _local="${_spec%%:*}"
  local _rest="${_spec#*:}"
  # If _rest contains a colon, it's "remoteHost:remotePort"; otherwise default host to localhost.
  if echo "$_rest" | grep -q ":"; then
    ssh -N -o ExitOnForwardFailure=yes -L "${_local}:${_rest}" "$_alias"
  else
    ssh -N -o ExitOnForwardFailure=yes -L "${_local}:localhost:${_rest}" "$_alias"
  fi
}`,
			Zsh: `stun() {
  if [[ "$1" == "--help" ]]; then
    echo "stun — Quick SSH tunnel shorthand"
    echo "Usage: stun <alias> <local:remote> or stun <alias> <local:remotehost:remote>"
    echo "Example: stun myserver 8080:80"
    echo "Example: stun myserver 5433:db.internal:5432"
    return 0
  fi
  if [[ $# -lt 2 ]]; then echo "stun: usage: stun <alias> <local:remote>" >&2; return 1; fi
  local _alias="$1"
  local _spec="$2"
  local _local="${_spec%%:*}"
  local _rest="${_spec#*:}"
  # If _rest contains a colon, it's "remoteHost:remotePort"; otherwise default host to localhost.
  if [[ "$_rest" == *:* ]]; then
    ssh -N -o ExitOnForwardFailure=yes -L "${_local}:${_rest}" "$_alias"
  else
    ssh -N -o ExitOnForwardFailure=yes -L "${_local}:localhost:${_rest}" "$_alias"
  fi
}`,
			Fish: `function stun
  if test "$argv[1]" = "--help"
    echo "stun — Quick SSH tunnel shorthand"
    echo "Usage: stun <alias> <local:remote> or stun <alias> <local:remotehost:remote>"
    echo "Example: stun myserver 8080:80"
    echo "Example: stun myserver 5433:db.internal:5432"
    return 0
  end
  if test (count $argv) -lt 2
    echo "stun: usage: stun <alias> <local:remote>" >&2; return 1
  end
  set -l _alias "$argv[1]"
  set -l _spec "$argv[2]"
  set -l _parts (string split ":" -- "$_spec")
  if test (count $_parts) -ge 3
    # local:remoteHost:remotePort — pass rest verbatim after local port
    set -l _local "$_parts[1]"
    set -l _rest (string join ":" -- $_parts[2..-1])
    ssh -N -o ExitOnForwardFailure=yes -L "$_local:$_rest" "$_alias"
  else
    # local:remotePort — default remote host to localhost
    ssh -N -o ExitOnForwardFailure=yes -L "$_parts[1]:localhost:$_parts[2]" "$_alias"
  end
end`,
		},
		{
			Name: "skey",
			Desc: "Copy default public key to clipboard",
			Bash: `skey() {
  if [ "$1" = "--help" ]; then
    echo "skey — Copy default public key to clipboard"
    echo "Usage: skey [keyfile]"
    echo "Example: skey ~/.ssh/id_ed25519.pub"
    return 0
  fi
  local keyfile="${1:-$HOME/.ssh/id_ed25519.pub}"
  if [ ! -f "$keyfile" ]; then keyfile="$HOME/.ssh/id_rsa.pub"; fi
  if [ ! -f "$keyfile" ]; then
    echo "skey: no public key found" >&2; return 1
  fi
  if command -v pbcopy >/dev/null 2>&1; then
    pbcopy < "$keyfile" && echo "Copied (pbcopy): $keyfile"
  elif command -v xclip >/dev/null 2>&1; then
    xclip -selection clipboard < "$keyfile" && echo "Copied (xclip): $keyfile"
  elif command -v xsel >/dev/null 2>&1; then
    xsel --clipboard --input < "$keyfile" && echo "Copied (xsel): $keyfile"
  else
    echo "skey: no clipboard tool found (pbcopy/xclip/xsel)" >&2; cat "$keyfile"; return 1
  fi
}`,
			Zsh: `skey() {
  if [[ "$1" == "--help" ]]; then
    echo "skey — Copy default public key to clipboard"
    echo "Usage: skey [keyfile]"
    echo "Example: skey ~/.ssh/id_ed25519.pub"
    return 0
  fi
  local keyfile="${1:-$HOME/.ssh/id_ed25519.pub}"
  if [[ ! -f "$keyfile" ]]; then keyfile="$HOME/.ssh/id_rsa.pub"; fi
  if [[ ! -f "$keyfile" ]]; then
    echo "skey: no public key found" >&2; return 1
  fi
  if command -v pbcopy >/dev/null 2>&1; then
    pbcopy < "$keyfile" && echo "Copied (pbcopy): $keyfile"
  elif command -v xclip >/dev/null 2>&1; then
    xclip -selection clipboard < "$keyfile" && echo "Copied (xclip): $keyfile"
  elif command -v xsel >/dev/null 2>&1; then
    xsel --clipboard --input < "$keyfile" && echo "Copied (xsel): $keyfile"
  else
    echo "skey: no clipboard tool found (pbcopy/xclip/xsel)" >&2; cat "$keyfile"; return 1
  fi
}`,
			Fish: `function skey
  if test "$argv[1]" = "--help"
    echo "skey — Copy default public key to clipboard"
    echo "Usage: skey [keyfile]"
    echo "Example: skey ~/.ssh/id_ed25519.pub"
    return 0
  end
  if test (count $argv) -gt 0
    set -l keyfile "$argv[1]"
  else
    set -l keyfile "$HOME/.ssh/id_ed25519.pub"
  end
  if not test -f "$keyfile"
    set keyfile "$HOME/.ssh/id_rsa.pub"
  end
  if not test -f "$keyfile"
    echo "skey: no public key found" >&2; return 1
  end
  if command -v pbcopy >/dev/null 2>&1
    pbcopy < "$keyfile"; and echo "Copied (pbcopy): $keyfile"
  else if command -v xclip >/dev/null 2>&1
    xclip -selection clipboard < "$keyfile"; and echo "Copied (xclip): $keyfile"
  else if command -v xsel >/dev/null 2>&1
    xsel --clipboard --input < "$keyfile"; and echo "Copied (xsel): $keyfile"
  else
    echo "skey: no clipboard tool found (pbcopy/xclip/xsel)" >&2; cat "$keyfile"; return 1
  end
end`,
		},
		{
			Name: "menv",
			Desc: "Load active mine env profile into the current shell",
			Bash: `menv() {
  if [ "$1" = "--help" ]; then
    echo "menv — Load active mine env profile into the current shell"
    echo "Usage: menv"
    echo "Example: menv"
    return 0
  fi
  eval "$(mine env export)"
}`,
			Zsh: `menv() {
  if [[ "$1" == "--help" ]]; then
    echo "menv — Load active mine env profile into the current shell"
    echo "Usage: menv"
    echo "Example: menv"
    return 0
  fi
  eval "$(mine env export)"
}`,
			Fish: `function menv
  if test "$argv[1]" = "--help"
    echo "menv — Load active mine env profile into the current shell"
    echo "Usage: menv"
    echo "Example: menv"
    return 0
  end
  mine env export --shell fish | source
end`,
		},
	}
}

// FunctionsScript generates the shell functions script for the given shell.
func FunctionsScript(shellName string) (string, error) {
	if !ValidShell(shellName) {
		return "", ShellError(shellName)
	}

	funcs := Functions()
	var out string
	out += "# mine shell functions — https://mine.rwolfe.io\n"
	out += "# Generated by: mine shell init\n\n"

	for _, fn := range funcs {
		out += fmt.Sprintf("# %s — %s\n", fn.Name, fn.Desc)
		switch shellName {
		case Bash:
			out += fn.Bash + "\n\n"
		case Zsh:
			out += fn.Zsh + "\n\n"
		case Fish:
			out += fn.Fish + "\n\n"
		}
	}

	return out, nil
}
