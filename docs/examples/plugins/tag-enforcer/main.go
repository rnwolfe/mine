package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Invocation struct {
	ProtocolVersion string            `json:"protocol_version"`
	Type            string            `json:"type"`
	Stage           string            `json:"stage,omitempty"`
	Mode            string            `json:"mode,omitempty"`
	Event           string            `json:"event,omitempty"`
	Command         string            `json:"command,omitempty"`
	Context         *Context          `json:"context,omitempty"`
	Args            []string          `json:"args,omitempty"`
	Flags           map[string]string `json:"flags,omitempty"`
}

type Context struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Flags     map[string]string `json:"flags"`
	Result    interface{}       `json:"result,omitempty"`
	Timestamp string            `json:"timestamp"`
}

type Response struct {
	Status  string   `json:"status"`
	Context *Context `json:"context,omitempty"`
	Error   string   `json:"error,omitempty"`
	Code    string   `json:"code,omitempty"`
}

func writeError(msg, code string) {
	json.NewEncoder(os.Stderr).Encode(Response{
		Status: "error",
		Error:  msg,
		Code:   code,
	})
	os.Exit(1)
}

func main() {
	var inv Invocation
	if err := json.NewDecoder(os.Stdin).Decode(&inv); err != nil {
		writeError("failed to parse invocation: "+err.Error(), "PARSE_ERROR")
	}

	if !strings.HasPrefix(inv.ProtocolVersion, "1.") {
		writeError(
			fmt.Sprintf("unsupported protocol version: %s", inv.ProtocolVersion),
			"UNSUPPORTED_PROTOCOL",
		)
	}

	switch inv.Type {
	case "hook":
		handleHook(&inv)
	case "command":
		handleCommand(&inv)
	case "lifecycle":
		handleLifecycle(&inv)
	}
}

func handleHook(inv *Invocation) {
	ctx := inv.Context
	if ctx == nil {
		writeError("hook invocation missing context", "MISSING_CONTEXT")
	}

	switch inv.Stage {
	case "prevalidate":
		if ctx.Flags == nil {
			ctx.Flags = map[string]string{}
		}
		if _, ok := ctx.Flags["tags"]; !ok {
			ctx.Flags["tags"] = "untagged"
		}
		json.NewEncoder(os.Stdout).Encode(Response{
			Status:  "ok",
			Context: ctx,
		})

	case "postexec":
		json.NewEncoder(os.Stdout).Encode(Response{
			Status:  "ok",
			Context: ctx,
		})
	}
}

func handleCommand(inv *Invocation) {
	if inv.Command == "policy" {
		fmt.Println("Tag Enforcer Policy")
		fmt.Println("-------------------")
		fmt.Println("All todos must have at least one tag.")
		fmt.Println("Todos without tags get the default tag: untagged")
		fmt.Println()
		fmt.Println("This runs at the prevalidate stage, so tags are")
		fmt.Println("applied before any other hooks see the todo.")
	}
}

func handleLifecycle(inv *Invocation) {
	if inv.Event == "health" {
		json.NewEncoder(os.Stdout).Encode(Response{Status: "ok"})
	}
}
