package tenant

import "context"

type ctxKey struct{}

// WithContext 将租户 id 写入 context（供 Agent 工具与 RAG 使用）。
func WithContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, tenantID)
}

// FromContext 读取租户 id；缺省为 default。
func FromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	v = SanitizeID(v)
	if v == "" {
		return DefaultID
	}
	return v
}
