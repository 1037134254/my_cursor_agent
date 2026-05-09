package agent

// ParseToolCallForTest 暴露 parseToolCall，供 internal/agent_test 使用。
func ParseToolCallForTest(reply string) (name, args string, ok bool) {
	return parseToolCall(reply)
}
