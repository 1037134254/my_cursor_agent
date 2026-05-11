package auth

import (
	"log"
	"os"
	"strings"
)

// refreshStore 刷新令牌后端（MySQL 优先；不可用时降级内存）。
type refreshStore interface {
	newToken(UserRecord) (string, error)
	consume(string) (UserRecord, bool)
	revoke(string)
}

var active refreshStore = &memoryRefreshStore{}

// defaultMySQLDSN 未显式配置 AUTH_REFRESH_MYSQL_DSN 时的探测目标
// （对应仓库 README 中的本地 Docker：root@127.0.0.1:3306）。
const defaultMySQLDSN = "root:root@tcp(127.0.0.1:3306)/my_cursor?parseTime=true&loc=Local&charset=utf8mb4"

// InitRefreshStore 自动选择刷新令牌存储：
//  1. 读 AUTH_REFRESH_MYSQL_DSN；显式 "memory" 强制内存；
//  2. 否则用配置 DSN 或内置默认 DSN 先做一次短超时探测；
//  3. 探测通过则使用 MySQL（库不存在会自动创建），不通则降级到内存。
func InitRefreshStore() error {
	raw := strings.TrimSpace(os.Getenv("AUTH_REFRESH_MYSQL_DSN"))
	if strings.EqualFold(raw, "memory") {
		active = &memoryRefreshStore{}
		log.Println("auth: 刷新令牌存储=内存（AUTH_REFRESH_MYSQL_DSN=memory）")
		return nil
	}
	dsn := raw
	source := "AUTH_REFRESH_MYSQL_DSN"
	if dsn == "" {
		dsn = defaultMySQLDSN
		source = "默认 DSN"
	}
	if err := probeMySQL(dsn); err != nil {
		active = &memoryRefreshStore{}
		log.Printf("auth: 检测 MySQL 失败（%s）：%v；降级=内存（进程重启失效）", source, err)
		return nil
	}
	st, err := newMySQLRefreshStore(dsn)
	if err != nil {
		active = &memoryRefreshStore{}
		log.Printf("auth: 初始化 MySQL 表失败：%v；降级=内存", err)
		return nil
	}
	active = st
	log.Printf("auth: 检测到 MySQL 可用（%s），刷新令牌存储=MySQL", source)
	return nil
}

// NewRefreshToken 签发刷新令牌（opaque，仅存 SHA256 哈希到 MySQL）。
func NewRefreshToken(user UserRecord) (string, error) {
	return active.newToken(user)
}

// ConsumeRefreshToken 校验并消费刷新令牌（一次性轮换）。
func ConsumeRefreshToken(token string) (UserRecord, bool) {
	return active.consume(token)
}

// RevokeRefreshToken 登出时作废。
func RevokeRefreshToken(token string) {
	active.revoke(token)
}
