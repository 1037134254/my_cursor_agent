package api

import (
	"errors"
	"net/http"
	"strings"

	"my_cursor/internal/auth"
	"my_cursor/internal/llm"
	"my_cursor/internal/tenant"

	"github.com/gin-gonic/gin"
)

type bindingView struct {
	TenantID    string `json:"tenant_id"`
	ProfileID   string `json:"profile_id"`
	Purpose     string `json:"purpose"`
	IsDefault   bool   `json:"is_default"`
	RPMOverride int    `json:"rpm_override"`
	CreatedAt   int64  `json:"created_at"`
}

type bindingUpsertItem struct {
	ProfileID   string `json:"profile_id"`
	Purpose     string `json:"purpose"`
	IsDefault   bool   `json:"is_default"`
	RPMOverride int    `json:"rpm_override"`
}

type bindingsUpsertReq struct {
	Bindings []bindingUpsertItem `json:"bindings"`
}

func toBindingView(b llm.TenantBinding) bindingView {
	return bindingView{
		TenantID:    b.TenantID,
		ProfileID:   b.ProfileID,
		Purpose:     string(b.Purpose),
		IsDefault:   b.IsDefault,
		RPMOverride: b.RPMOverride,
		CreatedAt:   b.CreatedAt.Unix(),
	}
}

// canAccessTenantBinding admin 可看/改任意租户；user 只能看自己的，不能改。
func canAccessTenantBinding(c *gin.Context, targetTID string, write bool) (bool, string) {
	p := PrincipalFrom(c)
	tid := tenant.SanitizeID(targetTID)
	if tid == "" {
		return false, "非法 tenant id"
	}
	if write {
		if !p.HasRole(auth.RoleAdmin) {
			return false, "只有 admin 可修改租户绑定"
		}
		return true, tid
	}
	if p.HasRole(auth.RoleAdmin) || p.TenantID == tid {
		return true, tid
	}
	return false, "只允许查看本租户的绑定"
}

// TenantModelsListHandler GET /api/tenants/:tid/models
func TenantModelsListHandler(c *gin.Context) {
	ok, tid := canAccessTenantBinding(c, c.Param("tid"), false)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": tid})
		return
	}
	bs, err := llm.Reg().Store().ListBindings(tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]bindingView, 0, len(bs))
	for _, b := range bs {
		out = append(out, toBindingView(b))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "tenant_id": tid, "bindings": out})
}

// TenantModelsPutHandler PUT /api/tenants/:tid/models
// 完整替换该租户的 binding；每个 purpose 仅允许一个 is_default=1。
func TenantModelsPutHandler(c *gin.Context) {
	ok, tid := canAccessTenantBinding(c, c.Param("tid"), true)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": tid})
		return
	}
	var req bindingsUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	bs := make([]llm.TenantBinding, 0, len(req.Bindings))
	for _, item := range req.Bindings {
		purpose := llm.Purpose(item.Purpose)
		if !llm.ValidPurpose(purpose) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的 purpose: " + item.Purpose})
			return
		}
		if strings.TrimSpace(item.ProfileID) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "profile_id 不能为空"})
			return
		}
		if item.RPMOverride < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rpm_override 不可为负"})
			return
		}
		bs = append(bs, llm.TenantBinding{
			TenantID:    tid,
			ProfileID:   item.ProfileID,
			Purpose:     purpose,
			IsDefault:   item.IsDefault,
			RPMOverride: item.RPMOverride,
		})
	}
	if err := llm.Reg().Store().SetBindings(tid, bs); err != nil {
		if errors.Is(err, llm.ErrReadOnly) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registry 只读"})
			return
		}
		if errors.Is(err, llm.ErrProfileNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	llm.Reg().EvictAll()
	Audit(c, "tenant_model_bind", tid)

	bsNow, _ := llm.Reg().Store().ListBindings(tid)
	out := make([]bindingView, 0, len(bsNow))
	for _, b := range bsNow {
		out = append(out, toBindingView(b))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "tenant_id": tid, "bindings": out})
}

// LLMCurrentHandler GET /api/llm/current?purpose=chat
// 返回当前登录用户所在租户、给定 purpose 的默认 profile（脱敏，不含 key）。
func LLMCurrentHandler(c *gin.Context) {
	p := PrincipalFrom(c)
	purpose := llm.Purpose(strings.TrimSpace(c.DefaultQuery("purpose", "chat")))
	if !llm.ValidPurpose(purpose) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的 purpose"})
		return
	}
	b, prof, err := llm.Reg().Store().DefaultBinding(p.TenantID, purpose)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     err.Error(),
			"tenant_id": p.TenantID,
			"purpose":   string(purpose),
			"hint":      "请联系 admin 通过 PUT /api/tenants/" + p.TenantID + "/models 绑定 profile",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"binding": toBindingView(b),
		"profile": toProfileView(prof),
	})
}
