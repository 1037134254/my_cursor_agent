package llm

import "time"

// 以下符号供 internal/llm_test 使用，勿在生产路径依赖其稳定性。

// RPMLimiter 为 rpmLimiter 的导出别名，便于外部测试包持有 nil 或真实限流器。
type RPMLimiter = rpmLimiter

// NewGatewayForTest 使用自定义 Provider 构造 Gateway（桩测试）；limiter 可为 nil。
func NewGatewayForTest(prov Provider, timeout time.Duration, maxRetries int, backoff time.Duration, limiter *RPMLimiter) *Gateway {
	return &Gateway{
		prov:       prov,
		timeout:    timeout,
		maxRetries: maxRetries,
		backoff:    backoff,
		limiter:    limiter,
	}
}

// IsRetryableForTest 暴露 isRetryable 判定逻辑。
func IsRetryableForTest(err error) bool { return isRetryable(err) }

// ParseChatContentForTest 暴露非流式响应体解析。
func ParseChatContentForTest(body []byte) (string, error) { return parseChatContent(body) }

// ParseStreamChunkForTest 暴露流式 chunk 行解析。
func ParseStreamChunkForTest(payload string) (delta string, streamErr error, err error) {
	return parseStreamChunk(payload)
}

// NewOpenAICompatProviderForTest 构造 OpenAI 兼容 Provider。
func NewOpenAICompatProviderForTest(cfg Config) Provider { return newOpenAICompatProvider(cfg) }

// NewRPMLimiterForTest 构造 RPM 限流器（≤0 返回 nil）。
func NewRPMLimiterForTest(maxPerMinute int) *RPMLimiter { return newRPMLimiter(maxPerMinute) }
