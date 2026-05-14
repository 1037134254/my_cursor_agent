package agent

import (
	"context"
	"fmt"
	"my_cursor/internal/llm"
	"my_cursor/internal/memory"
	"my_cursor/internal/rag"
	"my_cursor/internal/tenant"
	"my_cursor/internal/tool"
	"os"
	"strconv"
	"strings"
)

const SYSTEM_PROMPT = `
你是本地AI代码助手，支持多轮连续对话与工具调用。
若提示中包含「检索到的相关代码片段」，请优先结合这些片段回答，并注明引用路径。
当需要调用工具时，只输出一行命令，不要解释：
[TOOL:read_file]相对路径
[TOOL:write_file]相对路径||文件内容
当不需要工具时，直接输出最终答案。
`

type ToolFunc func(context.Context, string) (string, error)

type Agent struct {
	tools map[string]ToolFunc
}

var defaultAgent = newDefaultAgent()

func newDefaultAgent() *Agent {
	a := &Agent{
		tools: map[string]ToolFunc{},
	}
	a.RegisterTool("read_file", func(ctx context.Context, args string) (string, error) {
		tid := tenant.FromContext(ctx)
		content, err := tool.ReadFile(tid, strings.TrimSpace(args))
		if err != nil {
			return "", err
		}
		return content, nil
	})
	a.RegisterTool("write_file", func(ctx context.Context, args string) (string, error) {
		parts := strings.SplitN(args, "||", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("write_file 参数格式错误，期望 path||content")
		}
		path := strings.TrimSpace(parts[0])
		content := parts[1]
		tid := tenant.FromContext(ctx)
		if err := tool.WriteFile(tid, path, content); err != nil {
			return "", err
		}
		return "已写入文件: " + path, nil
	})
	return a
}

func (a *Agent) RegisterTool(name string, fn ToolFunc) {
	a.tools[name] = fn
}

func Chat(sessionID, msg, tenantID string) (reply string, outSessionID string, err error) {
	return defaultAgent.Chat(sessionID, msg, tenantID)
}

func (a *Agent) Chat(sessionID, msg, tenantID string) (reply string, outSessionID string, err error) {
	sessionID = memory.EnsureSessionID(tenantID, sessionID)
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "", sessionID, fmt.Errorf("msg 不能为空")
	}
	ctx := tenant.WithContext(context.Background(), tenantID)
	memory.Append(tenantID, sessionID, memory.Message{Role: "user", Content: msg})

	tasks := decomposeTasks(msg)
	finalParts := make([]string, 0, len(tasks))
	for _, task := range tasks {
		part, err := a.runSingleTask(ctx, sessionID, task)
		if err != nil {
			return "", sessionID, err
		}
		finalParts = append(finalParts, part)
	}

	finalReply := strings.Join(finalParts, "\n\n")
	memory.Append(tenantID, sessionID, memory.Message{Role: "assistant", Content: finalReply})
	return finalReply, sessionID, nil
}

// ChatStream 支持多轮会话；若需工具调用，先自动执行，再以流式方式输出最终答案。
func ChatStream(ctx context.Context, sessionID, msg, tenantID string, onDelta func(string) error) (outSessionID string, err error) {
	return defaultAgent.ChatStream(ctx, sessionID, msg, tenantID, onDelta)
}

func (a *Agent) ChatStream(ctx context.Context, sessionID, msg, tenantID string, onDelta func(string) error) (outSessionID string, err error) {
	sessionID = memory.EnsureSessionID(tenantID, sessionID)
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return sessionID, fmt.Errorf("msg 不能为空")
	}
	ctx = tenant.WithContext(ctx, tenantID)
	memory.Append(tenantID, sessionID, memory.Message{Role: "user", Content: msg})

	tasks := decomposeTasks(msg)
	for i, task := range tasks {
		prefix := ""
		if len(tasks) > 1 {
			prefix = fmt.Sprintf("\n[子任务 %d/%d]\n", i+1, len(tasks))
		}
		if prefix != "" {
			if err := onDelta(prefix); err != nil {
				return sessionID, err
			}
		}
		part, err := a.runSingleTask(ctx, sessionID, task)
		if err != nil {
			return sessionID, err
		}
		if err := onDelta(part); err != nil {
			return sessionID, err
		}
		if i < len(tasks)-1 {
			if err := onDelta("\n"); err != nil {
				return sessionID, err
			}
		}
	}
	return sessionID, nil
}

func (a *Agent) runSingleTask(ctx context.Context, sessionID, task string) (string, error) {
	const maxSteps = 6
	for i := 0; i < maxSteps; i++ {
		prompt := buildPrompt(ctx, sessionID, task)
		reply, err := llm.Chat(ctx, prompt)
		if err != nil {
			return "", err
		}
		reply = strings.TrimSpace(reply)

		name, args, ok := parseToolCall(reply)
		if !ok {
			return reply, nil
		}
		fn, exists := a.tools[name]
		if !exists {
			memory.Append(tenant.FromContext(ctx), sessionID, memory.Message{Role: "user", Content: "工具执行失败: 未知工具 " + name})
			continue
		}
		result, err := fn(ctx, args)
		if err != nil {
			memory.Append(tenant.FromContext(ctx), sessionID, memory.Message{Role: "user", Content: "工具执行失败: " + err.Error()})
			continue
		}
		memory.Append(tenant.FromContext(ctx), sessionID, memory.Message{Role: "user", Content: "工具执行结果: " + result})
	}
	return "", fmt.Errorf("自动任务执行超过最大步数")
}

// 一次用户请求对应一个任务（不再按分号/换行拆成多段）。
func decomposeTasks(msg string) []string {
	return []string{msg}
}

func buildPrompt(ctx context.Context, sessionID, task string) string {
	history := memory.GetRecent(tenant.FromContext(ctx), sessionID, 12)
	var b strings.Builder
	b.WriteString(SYSTEM_PROMPT)
	b.WriteString("\n已注册工具: read_file, write_file\n")
	b.WriteString("你可以多步思考并自动调用工具，完成后给出最终答案。\n")
	if block := retrieveCodeContext(ctx, task); block != "" {
		b.WriteString(block)
	}
	b.WriteString("\n近期会话上下文:\n")
	for _, m := range history {
		role := "用户"
		if m.Role == "assistant" {
			role = "助手"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	b.WriteString("\n当前任务: ")
	b.WriteString(task)
	return b.String()
}

func retrieveCodeContext(ctx context.Context, task string) string {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("RAG_CHAT_ENABLED")), "false") {
		return ""
	}
	svc, err := rag.Get()
	if err != nil {
		return ""
	}
	limit := uint64(8)
	if s := strings.TrimSpace(os.Getenv("RAG_TOP_K")); s != "" {
		if n, err := strconv.ParseUint(s, 10, 64); err == nil && n > 0 && n <= 32 {
			limit = n
		}
	}
	hits, err := svc.Search(ctx, tenant.FromContext(ctx), task, limit)
	if err != nil || len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n--- 检索到的相关代码片段（向量库，请结合回答）---\n")
	for i, h := range hits {
		src := h.Source
		if src == "" {
			src = "(unknown)"
		}
		kind := h.Kind
		if kind == "" {
			kind = "doc"
		}
		b.WriteString(fmt.Sprintf("\n[%d] score=%.3f kind=%s path=%s\n%s\n", i+1, h.Score, kind, src, h.Text))
	}
	b.WriteString("--- 片段结束 ---\n")
	return b.String()
}

func parseToolCall(reply string) (name, args string, ok bool) {
	reply = strings.TrimSpace(reply)
	if strings.HasPrefix(reply, "[READ]") {
		return "read_file", strings.TrimSpace(strings.TrimPrefix(reply, "[READ]")), true
	}
	if strings.HasPrefix(reply, "[WRITE]") {
		return "write_file", strings.TrimSpace(strings.TrimPrefix(reply, "[WRITE]")), true
	}
	if !strings.HasPrefix(reply, "[TOOL:") {
		return "", "", false
	}
	end := strings.Index(reply, "]")
	if end <= len("[TOOL:") {
		return "", "", false
	}
	name = strings.TrimSpace(reply[len("[TOOL:"):end])
	args = strings.TrimSpace(reply[end+1:])
	if name == "" {
		return "", "", false
	}
	return name, args, true
}
