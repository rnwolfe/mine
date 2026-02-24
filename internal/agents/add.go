package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// nameRe is the pattern for valid content names:
// lowercase letters/digits/hyphens, must start with a letter, 1-64 chars.
var nameRe = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,62}[a-z0-9])?$`)

// AddSkillResult describes the outcome of AddSkill.
type AddSkillResult struct {
	Dir     string // path to the new skill directory
	SKILLMD string // path to the generated SKILL.md
}

// AddCommandResult describes the outcome of AddCommand.
type AddCommandResult struct {
	File string // path to the new command markdown file
}

// AddAgentResult describes the outcome of AddAgent.
type AddAgentResult struct {
	File string // path to the new agent definition file
}

// AddRuleResult describes the outcome of AddRule.
type AddRuleResult struct {
	File string // path to the new rule file
}

// ValidateName returns an error if name is not a valid content name.
// Valid names are 1-64 chars, start with a lowercase letter, and contain
// only lowercase letters, digits, and hyphens (no leading/trailing hyphens;
// consecutive hyphens are not prohibited — only the regex above applies).
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 64 {
		return fmt.Errorf("name %q is too long (max 64 chars)", name)
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("name %q is invalid — must start with a lowercase letter and contain only lowercase letters, digits, and hyphens (pattern: %s)", name, nameRe.String())
	}
	return nil
}

// AddSkill scaffolds a new Agent Skill directory in the canonical store.
//
// The directory structure created is:
//
//	skills/<name>/
//	├── SKILL.md
//	├── scripts/
//	├── references/
//	└── assets/
//
// Returns an error if the skill already exists.
func AddSkill(name string) (*AddSkillResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	skillDir := filepath.Join(Dir(), "skills", name)
	if err := checkNotExists(skillDir); err != nil {
		return nil, fmt.Errorf("skill %q already exists at %s", name, skillDir)
	}

	// Create skill directory and subdirs.
	subdirs := []string{"", "scripts", "references", "assets"}
	for _, sub := range subdirs {
		p := filepath.Join(skillDir, sub)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", p, err)
		}
	}

	// Write SKILL.md template.
	skillMDPath := filepath.Join(skillDir, "SKILL.md")
	content := buildSkillMD(name)
	if err := os.WriteFile(skillMDPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("creating SKILL.md: %w", err)
	}

	return &AddSkillResult{
		Dir:     skillDir,
		SKILLMD: skillMDPath,
	}, nil
}

// AddCommand creates a new custom command markdown file in the canonical store.
//
// The file created is:
//
//	commands/<name>.md
//
// Returns an error if the command file already exists.
func AddCommand(name string) (*AddCommandResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	cmdFile := filepath.Join(Dir(), "commands", name+".md")
	if err := checkNotExists(cmdFile); err != nil {
		return nil, fmt.Errorf("command %q already exists at %s", name, cmdFile)
	}

	if err := os.MkdirAll(filepath.Dir(cmdFile), 0o755); err != nil {
		return nil, fmt.Errorf("creating commands directory: %w", err)
	}

	content := buildCommandMD(name)
	if err := os.WriteFile(cmdFile, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("creating command file: %w", err)
	}

	return &AddCommandResult{File: cmdFile}, nil
}

// AddAgent creates a new custom agent definition file in the canonical store.
//
// The file created is:
//
//	agents/<name>.md
//
// Returns an error if the agent file already exists.
func AddAgent(name string) (*AddAgentResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	agentFile := filepath.Join(Dir(), "agents", name+".md")
	if err := checkNotExists(agentFile); err != nil {
		return nil, fmt.Errorf("agent %q already exists at %s", name, agentFile)
	}

	if err := os.MkdirAll(filepath.Dir(agentFile), 0o755); err != nil {
		return nil, fmt.Errorf("creating agents directory: %w", err)
	}

	content := buildAgentMD(name)
	if err := os.WriteFile(agentFile, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("creating agent file: %w", err)
	}

	return &AddAgentResult{File: agentFile}, nil
}

// AddRule creates a new rule file in the canonical store.
//
// The file created is:
//
//	rules/<name>.md
//
// Returns an error if the rule file already exists.
func AddRule(name string) (*AddRuleResult, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	ruleFile := filepath.Join(Dir(), "rules", name+".md")
	if err := checkNotExists(ruleFile); err != nil {
		return nil, fmt.Errorf("rule %q already exists at %s", name, ruleFile)
	}

	if err := os.MkdirAll(filepath.Dir(ruleFile), 0o755); err != nil {
		return nil, fmt.Errorf("creating rules directory: %w", err)
	}

	content := buildRuleMD(name)
	if err := os.WriteFile(ruleFile, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("creating rule file: %w", err)
	}

	return &AddRuleResult{File: ruleFile}, nil
}

// checkNotExists returns an error if path already exists.
// It also propagates unexpected errors (e.g. permission denied, I/O error)
// so callers are not misled into thinking a path is safe to create.
func checkNotExists(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("already exists")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking path: %w", err)
	}
	return nil
}

// buildSkillMD generates the SKILL.md template content for the given name.
func buildSkillMD(name string) string {
	return fmt.Sprintf(`---
name: %s
description: >
  TODO: Describe what this skill does and when to use it.
---

## Instructions

TODO: Add step-by-step instructions for the agent.
`, name)
}

// buildCommandMD generates the command markdown template for the given name.
func buildCommandMD(name string) string {
	return fmt.Sprintf(`# %s

TODO: Describe what this command does and when to use it.

## Steps

1. TODO: Add step 1
2. TODO: Add step 2
`, name)
}

// buildAgentMD generates the agent definition template for the given name.
func buildAgentMD(name string) string {
	return fmt.Sprintf(`# %s

TODO: Describe this custom agent — its role, expertise, and when to invoke it.

## Instructions

TODO: Add detailed instructions for the agent.
`, name)
}

// buildRuleMD generates the rule file template for the given name.
func buildRuleMD(name string) string {
	return fmt.Sprintf(`# %s

TODO: Describe this rule and when it applies.

## Rule

TODO: Add the rule content here.
`, name)
}
