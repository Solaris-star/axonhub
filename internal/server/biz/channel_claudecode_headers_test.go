package biz

import (
	"strings"
	"testing"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/objects"
)

func ccBoolPtr(b bool) *bool { return &b }

func ccChannel(s *objects.ChannelSettings) *Channel {
	return &Channel{Channel: &ent.Channel{Settings: s}}
}

// TestClaudeCodeHeadersToggle verifies the per-channel ClaudeCodeHeaders toggle expands
// into the standard Claude Code CLI header set, that the channel's own header overrides
// still win on path collision (and compose), and that it is off by default.
func TestClaudeCodeHeadersToggle(t *testing.T) {
	find := func(ops []objects.OverrideOperation, path string) (string, bool) {
		var val string
		var ok bool
		for _, op := range ops {
			if op.Op == objects.OverrideOpSet && strings.EqualFold(op.Path, path) {
				val = op.Value
				ok = true
			}
		}
		return val, ok
	}

	// toggle on -> standard CC headers injected
	on := ccChannel(&objects.ChannelSettings{ClaudeCodeHeaders: ccBoolPtr(true)})
	ops := on.GetHeaderOverrideOperations()

	if ua, ok := find(ops, "User-Agent"); !ok || !strings.HasPrefix(ua, "claude-cli/") {
		t.Fatalf("expected claude-cli User-Agent, got %q (ok=%v)", ua, ok)
	}
	if beta, ok := find(ops, "Anthropic-Beta"); !ok || !strings.HasPrefix(beta, "claude-code-") {
		t.Fatalf("expected claude-code Anthropic-Beta, got %q", beta)
	}
	if app, ok := find(ops, "X-App"); !ok || app != "cli" {
		t.Fatalf("expected X-App=cli, got %q (ok=%v)", app, ok)
	}

	// user override wins on collision (User-Agent) and composes (extra header kept)
	custom := ccChannel(&objects.ChannelSettings{
		ClaudeCodeHeaders: ccBoolPtr(true),
		HeaderOverrideOperations: []objects.OverrideOperation{
			{Op: objects.OverrideOpSet, Path: "User-Agent", Value: "my-custom-ua"},
			{Op: objects.OverrideOpSet, Path: "X-Extra", Value: "yes"},
		},
	})
	cops := custom.GetHeaderOverrideOperations()
	if ua, _ := find(cops, "User-Agent"); ua != "my-custom-ua" {
		t.Fatalf("expected user override to win, got %q", ua)
	}
	if x, ok := find(cops, "X-Extra"); !ok || x != "yes" {
		t.Fatalf("expected user extra header preserved, got %q (ok=%v)", x, ok)
	}
	if _, ok := find(cops, "Anthropic-Beta"); !ok {
		t.Fatalf("expected standard Anthropic-Beta still present alongside user overrides")
	}

	// toggle off -> no CC headers
	if got := ccChannel(&objects.ChannelSettings{}).GetHeaderOverrideOperations(); len(got) != 0 {
		t.Fatalf("expected no override ops when toggle off, got %d", len(got))
	}

	// nil pointer -> off
	if got := ccChannel(&objects.ChannelSettings{ClaudeCodeHeaders: nil}).GetHeaderOverrideOperations(); len(got) != 0 {
		t.Fatalf("expected no override ops when toggle nil, got %d", len(got))
	}
}
