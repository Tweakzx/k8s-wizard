package workflow

import (
	"context"

	lgg "github.com/smallnest/langgraphgo/graph"
)

// RouteAfterParse determines the next node after parsing intent.
func RouteAfterParse(ctx context.Context, state AgentState) string {
	// If there was an error, end
	if state.Error != nil {
		return lgg.END
	}

	// If not a K8s operation, return the chat reply
	if !state.IsK8sOperation {
		return lgg.END
	}

	// If suggestions available, show them instead of form
	if len(state.Suggestions) > 0 {
		return "show_suggestions"
	}

	// Otherwise, proceed to merge form data
	return "merge_form"
}

// RouteAfterClarify determines the next node after checking clarification.
func RouteAfterClarify(ctx context.Context, state AgentState) string {
	// If clarification is needed, return to let user fill the form
	if state.NeedsClarification {
		state.Status = StatusNeedsInfo
		return lgg.END
	}

	// Otherwise, proceed to generate preview
	return "generate_preview"
}

// RouteAfterPreview determines the next node after generating preview.
func RouteAfterPreview(ctx context.Context, state AgentState) string {
	// If no preview was generated, the operation is unsupported
	if state.ActionPreview == nil {
		state.Status = StatusError
		state.Result = "❓ 抱歉，暂不支持此操作。\n\n**支持的操作:**\n• 部署应用: 部署一个 nginx\n• 查看资源: 查看所有 pod/deployment/service\n• 扩缩容: 把 nginx 扩容到 5 个\n• 删除资源: 删除名为 xxx 的 deployment"
		return lgg.END
	}

	// For get operations (read-only), execute directly without confirmation
	if state.Action.Action == "get" || state.Action.Action == "list" || state.Action.Action == "show" {
		return "execute"
	}

	// If user has already confirmed, execute
	if state.Confirm != nil && *state.Confirm {
		return "execute"
	}

	// Otherwise, wait for confirmation
	state.Status = StatusNeedsConfirm
	return lgg.END
}
