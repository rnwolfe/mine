// Package tips provides actionable tips for mine CLI discovery.
package tips

import "time"

// all is the full tip pool covering all major mine features.
var all = []string{
	"`mine proj add .` to register this directory as a project.",
	"`p <name>` to jump to any registered project instantly (requires shell helpers).",
	"`mine stash track ~/.zshrc` to version your dotfiles.",
	"`mine stash commit` to snapshot your tracked files.",
	"`cat main.go | mine ai ask \"explain this\"` to get AI help without leaving the terminal.",
	"`mine vault get TOKEN | xargs -I {} curl -H \"Authorization: {}\"` to pipe secrets directly to commands.",
	"`mine env inject -- npm run dev` to run commands with env vars injected.",
	"`mine craft dev go` to scaffold a new Go project with batteries included.",
	"`mine tmux layout save` to save your current tmux layout and restore it later.",
	"`mine hook create` to add a hook that runs before or after any mine command.",
	"`mine ai review` to get a quick AI code review of your staged changes.",
	"`mine ai commit` to auto-generate a commit message from your staged diff.",
	"`mine todo add \"idea\"` to capture a fleeting thought before it escapes.",
	"`mine todo done <id>` to check off a task — `mine todo` to see what's left.",
	"`mine proj` to fuzzy-pick and switch between all your registered projects.",
	"`mine vault set KEY VALUE` to store a secret, encrypted locally.",
	"`mine env set KEY=VALUE` to add a variable to your current env profile.",
	"`mine env switch prod` to activate a different env profile for this project.",
	"`mine git sweep` to delete all merged branches in one shot.",
	"`mine dig` to start a focused 25-minute Pomodoro session.",
	"`mine plugin install <name>` to extend mine with community plugins.",
	"`mine tmux new <name>` to create a named tmux session you can rejoin later.",
	"`pp` to jump back to your previous project (requires shell helpers).",
	"`mine hook list` to see all active hooks for the current command pipeline.",
	"`mine craft ci github` to add a GitHub Actions CI workflow to your project.",
}

// All returns all tips in the pool.
func All() []string {
	return all
}

// Daily returns a deterministic tip for the given day.
// The same tip is returned all day; it changes each day.
func Daily(t time.Time) string {
	dayOfYear := t.YearDay()
	return all[dayOfYear%len(all)]
}

// Random returns a tip based on the current time's minute,
// useful when you want variety within a day.
func Random(t time.Time) string {
	return all[t.Minute()%len(all)]
}
