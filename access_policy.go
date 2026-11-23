package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/server"
)

// accessPolicy holds deployment-level restrictions read from the environment.
//
//	ALLOWED_EMAILS: comma-separated Things Cloud account emails that may sign
//	in. When set, any other account is refused before its credentials are
//	sent to Things Cloud. Empty means no restriction (upstream behavior).
//
//	READ_ONLY: when "true" or "1", only tools annotated read-only are
//	registered, and every write to Things Cloud is refused.
//
//	WRITE_TOOLS: comma-separated names of the write tools to register, such
//	as "things_create_task,things_edit_item". When set, the read tools and
//	only these write tools are registered. It can't be combined with
//	READ_ONLY, and a name that isn't a write tool stops the server.
type accessPolicy struct {
	allowed    map[string]bool
	readOnly   bool
	writeTools map[string]bool // nil registers every write tool
}

func loadAccessPolicy() accessPolicy {
	p := newAccessPolicy(os.Getenv("ALLOWED_EMAILS"), os.Getenv("READ_ONLY"))
	p.writeTools = parseWriteTools(os.Getenv("WRITE_TOOLS"))
	return p
}

// parseWriteTools reads WRITE_TOOLS. Unset or blank is nil, every write tool.
func parseWriteTools(v string) map[string]bool {
	var names map[string]bool
	for _, name := range strings.Split(v, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if names == nil {
			names = make(map[string]bool)
		}
		names[name] = true
	}
	return names
}

func newAccessPolicy(allowedEmails, readOnly string) accessPolicy {
	p := accessPolicy{}
	for _, e := range strings.Split(allowedEmails, ",") {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if p.allowed == nil {
			p.allowed = make(map[string]bool)
		}
		p.allowed[e] = true
	}
	switch strings.ToLower(strings.TrimSpace(readOnly)) {
	case "1", "true", "yes":
		p.readOnly = true
	}
	return p
}

// checkEmail returns an error when an allowlist is configured and email is not on it.
func (p accessPolicy) checkEmail(email string) error {
	if p.allowed == nil {
		return nil
	}
	if p.allowed[strings.ToLower(strings.TrimSpace(email))] {
		return nil
	}
	return fmt.Errorf("this server does not accept that Things Cloud account")
}

var errReadOnly = fmt.Errorf("this server is read-only; writes to Things Cloud are disabled")

func isReadTool(tool server.ServerTool) bool {
	hint := tool.Tool.Annotations.ReadOnlyHint
	return hint != nil && *hint
}

// checkTools reports a WRITE_TOOLS that names something other than a write
// tool, or that's set along with READ_ONLY.
func (p accessPolicy) checkTools(tools []server.ServerTool) error {
	if p.writeTools == nil {
		return nil
	}
	if p.readOnly {
		return fmt.Errorf("READ_ONLY and WRITE_TOOLS can't both be set")
	}
	writes := make(map[string]bool)
	for _, tool := range tools {
		if !isReadTool(tool) {
			writes[tool.Tool.Name] = true
		}
	}
	var unknown []string
	for name := range p.writeTools {
		if !writes[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("WRITE_TOOLS names %s, which isn't a write tool", strings.Join(unknown, ", "))
	}
	return nil
}

// filterTools keeps the tools the policy registers.
func (p accessPolicy) filterTools(tools []server.ServerTool) []server.ServerTool {
	if !p.readOnly && p.writeTools == nil {
		return tools
	}
	kept := tools[:0]
	for _, tool := range tools {
		if isReadTool(tool) || (!p.readOnly && p.writeTools[tool.Tool.Name]) {
			kept = append(kept, tool)
		}
	}
	return kept
}
