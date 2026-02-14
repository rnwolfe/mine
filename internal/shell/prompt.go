package shell

// PromptScript generates the prompt integration script for the given shell.
// It defines a __mine_prompt helper that calls `mine status --json` and
// formats a prompt segment showing todo count and active dig sessions.
func PromptScript(shellName string) (string, error) {
	if !ValidShell(shellName) {
		return "", ShellError(shellName)
	}

	var out string
	out += "# mine prompt integration — https://mine.rwolfe.io\n"
	out += "# Shows todo count and dig streak in your prompt.\n"
	out += "# Works standalone or with Starship (see: mine shell prompt).\n\n"

	switch shellName {
	case Bash:
		out += bashPrompt()
	case Zsh:
		out += zshPrompt()
	case Fish:
		out += fishPrompt()
	}

	return out, nil
}

func bashPrompt() string {
	return `# Cache mine status for 30s to keep prompt fast.
__mine_prompt_cache=""
__mine_prompt_ts=0

__mine_prompt() {
  local now
  now=$(date +%s)
  if (( now - __mine_prompt_ts > 30 )); then
    __mine_prompt_cache=$(mine status --json 2>/dev/null)
    __mine_prompt_ts=$now
  fi

  if [ -z "$__mine_prompt_cache" ]; then return; fi

  local todos streak
  todos=$(echo "$__mine_prompt_cache" | grep -o '"open_todos":[0-9]*' | grep -o '[0-9]*')
  streak=$(echo "$__mine_prompt_cache" | grep -o '"dig_streak":[0-9]*' | grep -o '[0-9]*')

  local seg=""
  if [ -n "$todos" ] && [ "$todos" != "0" ]; then
    seg="${seg}${todos}t"
  fi
  if [ -n "$streak" ] && [ "$streak" != "0" ]; then
    [ -n "$seg" ] && seg="${seg}|"
    seg="${seg}${streak}d"
  fi

  if [ -n "$seg" ]; then
    printf "[%s] " "$seg"
  fi
}

# Append mine segment to PS1 if not already present.
if [[ "$PROMPT_COMMAND" != *"__mine_prompt"* ]]; then
  __mine_orig_ps1="$PS1"
  PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND;}"'PS1="$(__mine_prompt)${__mine_orig_ps1}"'
fi
`
}

func zshPrompt() string {
	return `# Cache mine status for 30s to keep prompt fast.
typeset -g __mine_prompt_cache=""
typeset -g __mine_prompt_ts=0

__mine_prompt() {
  local now=$(date +%s)
  if (( now - __mine_prompt_ts > 30 )); then
    __mine_prompt_cache=$(mine status --json 2>/dev/null)
    __mine_prompt_ts=$now
  fi

  if [[ -z "$__mine_prompt_cache" ]]; then return; fi

  local todos streak
  todos=$(echo "$__mine_prompt_cache" | grep -o '"open_todos":[0-9]*' | grep -o '[0-9]*')
  streak=$(echo "$__mine_prompt_cache" | grep -o '"dig_streak":[0-9]*' | grep -o '[0-9]*')

  local seg=""
  if [[ -n "$todos" && "$todos" != "0" ]]; then
    seg="${seg}${todos}t"
  fi
  if [[ -n "$streak" && "$streak" != "0" ]]; then
    [[ -n "$seg" ]] && seg="${seg}|"
    seg="${seg}${streak}d"
  fi

  if [[ -n "$seg" ]]; then
    printf "[%s] " "$seg"
  fi
}

# Add mine segment to prompt via precmd hook.
if (( ! ${+functions[__mine_precmd]} )); then
  __mine_precmd() { PROMPT="$(__mine_prompt)${PROMPT#\[[0-9tdTD|]*\] }" ; }
  autoload -Uz add-zsh-hook
  add-zsh-hook precmd __mine_precmd
fi
`
}

func fishPrompt() string {
	return `# Cache mine status for 30s to keep prompt fast.
set -g __mine_prompt_cache ""
set -g __mine_prompt_ts 0

function __mine_prompt
  set -l now (date +%s)
  if test (math "$now - $__mine_prompt_ts") -gt 30
    set -g __mine_prompt_cache (mine status --json 2>/dev/null)
    set -g __mine_prompt_ts $now
  end

  if test -z "$__mine_prompt_cache"; return; end

  set -l todos (echo "$__mine_prompt_cache" | grep -o '"open_todos":[0-9]*' | grep -o '[0-9]*')
  set -l streak (echo "$__mine_prompt_cache" | grep -o '"dig_streak":[0-9]*' | grep -o '[0-9]*')

  set -l seg ""
  if test -n "$todos" -a "$todos" != "0"
    set seg "$seg""$todos"t
  end
  if test -n "$streak" -a "$streak" != "0"
    if test -n "$seg"
      set seg "$seg""|"
    end
    set seg "$seg""$streak"d
  end

  if test -n "$seg"
    printf "[%s] " "$seg"
  end
end
`
}

// StarshipConfig returns a TOML snippet for Starship prompt integration.
func StarshipConfig() string {
	return `# Add this to your ~/.config/starship.toml
# Displays mine todo count and dig streak in your prompt.

[custom.mine]
command = "mine status --prompt"
when = "command -v mine"
shell = ["sh"]
format = "[$output]($style) "
style = "bold yellow"
`
}
