# Command Reference

## mine

The default command shows your personal dashboard.

```
$ mine
⛏ Hey Ryan!

  📋 Todos      3 open (1 overdue!)
  📅 Today      Friday, February 14
  ⚙️  Mine      0.1.0

  tip: `mine todo` to tackle that overdue task.
```

If mine hasn't been initialized, it prompts you to run `mine init`.

---

## mine init

Guided first-time setup. Creates config and data directories.

```bash
mine init
```

Auto-detects your name from `~/.gitconfig`. Creates:
- `~/.config/mine/config.toml`
- `~/.local/share/mine/mine.db`

---

## mine todo

Fast task management with priorities, due dates, and tags.

### List todos

```bash
mine todo         # show open todos
mine todo --all   # include completed
mine t            # alias
```

### Add a todo

```bash
mine todo add "build the feature"
mine todo add "fix bug" -p high -d tomorrow
mine todo add "write docs" -p low -d 2026-03-01 --tags "docs,v0.2"
```

**Priorities**: `low` (or `l`), `med` (default), `high` (or `h`), `crit` (or `c`, `!`)

**Due dates**: `today`, `tomorrow` (or `tom`), `next-week` (or `nw`), `next-month` (or `nm`), or `YYYY-MM-DD`

### Complete a todo

```bash
mine todo done 1     # mark #1 as done
mine todo do 1       # alias
mine todo x 1        # alias
```

### Delete a todo

```bash
mine todo rm 1       # delete #1
mine todo remove 1   # alias
mine todo delete 1   # alias
```

### Edit a todo

```bash
mine todo edit 1 "new title"
```

---

## mine stash

Track, diff, and manage your dotfiles.

### Initialize

```bash
mine stash init
```

Creates a stash directory at `~/.local/share/mine/stash/`.

### Track a file

```bash
mine stash track ~/.zshrc
mine stash track ~/.gitconfig
mine stash track ~/.config/starship.toml
```

### List tracked files

```bash
mine stash list
```

### Check for changes

```bash
mine stash diff
```

Shows which tracked files have been modified since last stash.

---

## mine craft

Scaffold projects and bootstrap dev tool configurations.

### Bootstrap a project

```bash
mine craft dev go       # Go project with module, main.go, Makefile
mine craft dev node     # Node.js with package.json
mine craft dev python   # Python with pyproject.toml and .venv
```

### Set up git

```bash
mine craft git          # git init + .gitignore
```

---

## mine dig

Pomodoro-style focus timer with streak tracking.

### Start a session

```bash
mine dig          # 25 minutes (default)
mine dig 45m      # 45 minutes
mine dig 1h       # 1 hour
```

Press `Ctrl+C` to end early. Sessions over 5 minutes still count.

### View stats

```bash
mine dig stats
```

Shows current streak, longest streak, and total deep work time.

---

## mine shell

Shell integration and enhancements.

### Generate completions

```bash
mine shell completions         # auto-detect shell
mine shell completions zsh
mine shell completions bash
mine shell completions fish
```

### Show recommended aliases

```bash
mine shell aliases
```

Outputs:
```
alias m='mine'
alias mt='mine todo'
alias mta='mine todo add'
alias mtd='mine todo done'
alias md='mine dig'
alias mc='mine craft'
alias ms='mine stash'
```

---

## mine config

View and manage configuration.

### Show config

```bash
mine config
```

### Show config file path

```bash
mine config path
```

### Edit directly

```bash
$EDITOR $(mine config path)
```

---

## mine version

Print version, commit hash, and build date.

```bash
mine version
```
