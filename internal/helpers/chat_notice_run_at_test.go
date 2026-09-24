package helpers

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/corecmd"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/testseam"
	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/pkg/edition"
)

type noticeRunAtTestCall struct {
	server string
	tool   string
	args   map[string]any
}

type noticeRunAtTestCaller struct {
	calls []noticeRunAtTestCall
}

func (c *noticeRunAtTestCaller) CallTool(_ context.Context, server, tool string, args map[string]any) (*edition.ToolResult, error) {
	c.calls = append(c.calls, noticeRunAtTestCall{server: server, tool: tool, args: args})
	return &edition.ToolResult{Content: []edition.ContentBlock{{Type: "text", Text: "{}"}}}, nil
}

func (c *noticeRunAtTestCaller) Format() string { return "json" }
func (c *noticeRunAtTestCaller) DryRun() bool   { return false }
func (*noticeRunAtTestCaller) Fields() string   { return "" }
func (*noticeRunAtTestCaller) JQ() string       { return "" }

// TestCrossPlatformCoverageNormalizeNoticeRunAtText covers the runAtText
// wire-format normalization for `chat group notice create --run-at`
// (Issue #1438: ISO-8601 inputs previously passed through unconverted and
// were rejected by the backend).
func TestCrossPlatformCoverageNormalizeNoticeRunAtText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "wire format passthrough", input: "2026-07-03 09:00:00", want: "2026-07-03 09:00:00"},
		{name: "iso8601 with offset", input: "2026-07-03T09:00:00+08:00", want: "2026-07-03 09:00:00"},
		{name: "utc converted to shanghai", input: "2026-07-03T01:00:00Z", want: "2026-07-03 09:00:00"},
		{name: "other offset converted to shanghai", input: "2026-07-03T02:00:00+01:00", want: "2026-07-03 09:00:00"},
		{name: "whitespace trimmed", input: "  2026-07-03 09:00:00  ", want: "2026-07-03 09:00:00"},
		{name: "date only rejected", input: "2026-07-03", wantErr: true},
		{name: "garbage rejected", input: "明早九点", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeNoticeRunAtText(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("normalizeNoticeRunAtText(%q) err = nil, want validation error", test.input)
				}
				if !strings.Contains(err.Error(), "invalid --run-at format") {
					t.Fatalf("normalizeNoticeRunAtText(%q) err = %v, want validation-classified message", test.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeNoticeRunAtText(%q) returned error: %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("normalizeNoticeRunAtText(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

// TestCrossPlatformCoverageChatGroupNoticeCreateNormalizesRunAt asserts the
// RunE path converts --run-at into the backend yyyy-MM-dd HH:mm:ss wire
// format before calling create_group_notice (Issue #1438).
func TestCrossPlatformCoverageChatGroupNoticeCreateNormalizesRunAt(t *testing.T) {
	testseam.Protect(t, &deps)
	caller := &noticeRunAtTestCaller{}
	InitDeps(caller)
	deps.Out.w = io.Discard
	deps.Out.errW = io.Discard

	root := newChatCommand()
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{
		"group", "notice", "create",
		"--conversation-id", "cid-1", "--content", "hello",
		"--run-at", "2026-07-03T09:00:00+08:00",
	})
	if err := corecmd.ExecuteForTest(root); err != nil {
		t.Fatalf("notice create returned error: %v", err)
	}
	if len(caller.calls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(caller.calls))
	}
	call := caller.calls[0]
	if call.server != "im" || call.tool != "create_group_notice" {
		t.Fatalf("tool = %s/%s, want im/create_group_notice", call.server, call.tool)
	}
	if call.args["scheduled"] != true {
		t.Fatalf("scheduled = %#v, want true", call.args["scheduled"])
	}
	if call.args["runAtText"] != "2026-07-03 09:00:00" {
		t.Fatalf("runAtText = %#v, want 2026-07-03 09:00:00", call.args["runAtText"])
	}
}

// TestCrossPlatformCoverageChatGroupNoticeCreateRejectsBadRunAt keeps the
// invalid-input path classified as validation (exit code 3 boundary).
func TestCrossPlatformCoverageChatGroupNoticeCreateRejectsBadRunAt(t *testing.T) {
	testseam.Protect(t, &deps)
	caller := &noticeRunAtTestCaller{}
	InitDeps(caller)
	deps.Out.w = io.Discard
	deps.Out.errW = io.Discard

	root := newChatCommand()
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{
		"group", "notice", "create",
		"--conversation-id", "cid-1", "--content", "hello",
		"--run-at", "2026-07-03",
	})
	err := corecmd.ExecuteForTest(root)
	if err == nil {
		t.Fatal("notice create with date-only --run-at returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "invalid --run-at format") {
		t.Fatalf("err = %v, want invalid --run-at format message", err)
	}
	if len(caller.calls) != 0 {
		t.Fatalf("tool calls = %d, want 0 (must not call backend on invalid input)", len(caller.calls))
	}
}
