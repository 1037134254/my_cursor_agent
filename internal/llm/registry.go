package llm

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// Registry 多租户模型注册表：profile 持久化 + per-(profile, rpm_override) 的 Gateway 缓存。
type Registry struct {
	mu       sync.RWMutex
	store    ProfileStore
	gateways map[string]*Gateway // key = profile_id + "|" + rpm_override
}

var (
	defaultRegistry *Registry
	regMu           sync.RWMutex
)

// InitRegistry 启动时初始化：优先用 MySQL；探测失败或显式 LLM_REGISTRY=env 时降级为只读 env 模式。
// 该方法是幂等的（重复调用会重建 store；推理可继续）。
func InitRegistry() {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_REGISTRY")))
	if mode == "env" {
		setRegistry(&Registry{store: envProfileStore{}, gateways: map[string]*Gateway{}})
		log.Println("llm: 模型注册表=env（LLM_REGISTRY=env，所有租户共享 .env 中的 LLM_* 配置）")
		return
	}
	dsn := strings.TrimSpace(os.Getenv("MODEL_REGISTRY_MYSQL_DSN"))
	source := "MODEL_REGISTRY_MYSQL_DSN"
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("AUTH_REFRESH_MYSQL_DSN"))
		source = "AUTH_REFRESH_MYSQL_DSN"
	}
	if dsn == "" {
		setRegistry(&Registry{store: envProfileStore{}, gateways: map[string]*Gateway{}})
		log.Println("llm: 未配置 MYSQL DSN，模型注册表=env（只读，建议生产配置 MySQL）")
		return
	}
	if err := probeRegistryMySQL(dsn); err != nil {
		setRegistry(&Registry{store: envProfileStore{}, gateways: map[string]*Gateway{}})
		log.Printf("llm: 检测 MySQL 失败（%s）：%v；模型注册表降级=env", source, err)
		return
	}
	st, err := newMySQLProfileStore(dsn)
	if err != nil {
		setRegistry(&Registry{store: envProfileStore{}, gateways: map[string]*Gateway{}})
		log.Printf("llm: 初始化 MySQL profile 表失败：%v；模型注册表降级=env", err)
		return
	}
	setRegistry(&Registry{store: st, gateways: map[string]*Gateway{}})

	profs, err := st.ListProfiles()
	if err != nil {
		log.Printf("llm: 列出 profile 失败：%v", err)
		return
	}
	if len(profs) == 0 {
		log.Printf("llm: 模型注册表=MySQL（%s），但表 model_profile 为空；请用 admin 调用 POST /api/llm/profiles 创建首个 profile 后再使用 /api/chat", source)
	} else {
		log.Printf("llm: 模型注册表=MySQL（%s），已加载 %d 条 profile", source, len(profs))
	}
}

func setRegistry(r *Registry) {
	regMu.Lock()
	defer regMu.Unlock()
	defaultRegistry = r
}

// Reg 返回当前注册表（启动前调用会返回 env 兜底实例，避免空指针）。
func Reg() *Registry {
	regMu.RLock()
	r := defaultRegistry
	regMu.RUnlock()
	if r != nil {
		return r
	}
	tmp := &Registry{store: envProfileStore{}, gateways: map[string]*Gateway{}}
	regMu.Lock()
	if defaultRegistry == nil {
		defaultRegistry = tmp
	}
	r = defaultRegistry
	regMu.Unlock()
	return r
}

// Store 暴露底层持久化（供 API 层 CRUD）。
func (r *Registry) Store() ProfileStore { return r.store }

// gatewayKey 不同 RPM 用不同 limiter，所以 cache key 把 rpm_override 也带上。
func gatewayKey(profileID string, rpmOverride int) string {
	return fmt.Sprintf("%s|%d", profileID, rpmOverride)
}

// GatewayFor 根据 (tenantID, purpose) 找到默认绑定，构造（或复用）Gateway。
func (r *Registry) GatewayFor(tenantID string, purpose Purpose) (*Gateway, Profile, error) {
	b, prof, err := r.store.DefaultBinding(tenantID, purpose)
	if err != nil {
		return nil, Profile{}, err
	}
	return r.gatewayFromProfile(prof, b.RPMOverride), prof, nil
}

func (r *Registry) gatewayFromProfile(prof Profile, rpmOverride int) *Gateway {
	key := gatewayKey(prof.ID, rpmOverride)
	r.mu.RLock()
	gw := r.gateways[key]
	r.mu.RUnlock()
	if gw != nil {
		return gw
	}
	cfg := prof.EffectiveConfig(rpmOverride)
	cfg.APIKey = resolveAPIKey(prof)
	gw = NewGateway(cfg)

	r.mu.Lock()
	if existed, ok := r.gateways[key]; ok {
		r.mu.Unlock()
		return existed
	}
	r.gateways[key] = gw
	r.mu.Unlock()
	return gw
}

// resolveAPIKey env 模式直接复用 LoadConfig().APIKey；mysql 模式从 prof.APIKeyRef 指定的 env 变量取真值。
func resolveAPIKey(p Profile) string {
	if p.APIKeyRef == "" {
		// env-default-chat 走这里：让 LoadConfig 把当前环境变量里的 key 算出来
		if p.ID == envProfileID {
			return LoadConfig().APIKey
		}
		return ""
	}
	return strings.TrimSpace(os.Getenv(p.APIKeyRef))
}

// EvictAll 清空 Gateway 缓存（profile / binding 写操作后调用，确保下次推理读最新配置）。
func (r *Registry) EvictAll() {
	r.mu.Lock()
	r.gateways = map[string]*Gateway{}
	r.mu.Unlock()
}

// EvictProfile 仅清掉某 profile 关联的 Gateway。
func (r *Registry) EvictProfile(profileID string) {
	r.mu.Lock()
	for k := range r.gateways {
		if strings.HasPrefix(k, profileID+"|") {
			delete(r.gateways, k)
		}
	}
	r.mu.Unlock()
}
