package models

import "time"

// 节点状态
const (
	NodeStatusOnline  = "online"
	NodeStatusOffline = "offline"
)

type Node struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	Name          string  `gorm:"type:varchar(100)" json:"name"`
	Region        string  `gorm:"type:varchar(50);index" json:"region"`
	Type          string  `gorm:"type:varchar(20);index" json:"type"`
	Status        string  `gorm:"type:varchar(20);default:'offline';index" json:"status"`
	Load          float64 `gorm:"default:0" json:"load"`
	Speed         float64 `gorm:"default:0" json:"speed"`
	Uptime        int     `gorm:"default:0" json:"uptime"`
	Latency       int     `gorm:"default:0" json:"latency"`
	Description   *string `gorm:"type:text" json:"description"`
	Config        *string `gorm:"type:text" json:"config"`
	IsRecommended bool    `gorm:"default:false" json:"is_recommended"`
	IsActive      bool    `gorm:"default:true;index" json:"is_active"`
	IsManual      bool    `gorm:"default:false" json:"is_manual"`
	// PinnedOnline 固定在线：节点只在部分地区可访问（例如仅中国境内可达的家庭宽带 IP），
	// 服务端探测必然失败，若按探测结果判离线就会被订阅过滤掉。勾选后不再被自动探测改写状态，
	// 且始终下发（见 NodeDeliverable）。
	PinnedOnline bool       `gorm:"default:false;index" json:"pinned_online"`
	SourceIndex  int        `gorm:"default:0" json:"source_index"`
	SourceURL    string     `gorm:"type:text" json:"source_url"`
	OrderIndex   int        `gorm:"default:0;index" json:"order_index"`
	LastTest     *time.Time `json:"last_test"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Node) TableName() string {
	return "nodes"
}

// NodeDeliverableWhere 返回「该下发给客户」的查询片段：
// 启用且（探测在线 或 管理员勾选了固定在线）。
//
// 为什么需要「固定在线」：部分节点只对特定地区开放（如仅中国境内可达的住宅 IP），
// 服务端探测永远失败，按探测结果过滤会让这些节点永远下发给不出去——
// 而客户在国内是能正常用的。
func NodeDeliverableWhere() (string, []interface{}) {
	return "is_active = ? AND (status = ? OR pinned_online = ?)", []interface{}{true, NodeStatusOnline, true}
}

type CustomNode struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Name             string     `gorm:"type:varchar(100)" json:"name"`
	DisplayName      string     `gorm:"type:varchar(100)" json:"display_name"`
	Protocol         string     `gorm:"type:varchar(20)" json:"protocol"`
	Domain           string     `gorm:"type:varchar(255)" json:"domain"`
	Port             int        `gorm:"default:443" json:"port"`
	Config           string     `gorm:"type:text" json:"config"`
	Status           string     `gorm:"type:varchar(20);default:'inactive'" json:"status"`
	IsActive         bool       `gorm:"default:true;index" json:"is_active"`
	Latency          int        `gorm:"default:0" json:"latency"`
	LastTest         *time.Time `json:"last_test"`
	ExpireTime       *time.Time `json:"expire_time"`
	FollowUserExpire bool       `gorm:"default:false" json:"follow_user_expire"`
	SourceURL        string     `gorm:"type:text;index" json:"source_url"` // 来源订阅地址（用于订阅更新追踪）
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CustomNode) TableName() string {
	return "custom_nodes"
}

type UserCustomNode struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index:idx_user_node" json:"user_id"`
	CustomNodeID  uint       `gorm:"index:idx_user_node" json:"custom_node_id"`
	ExpiresAt     *time.Time `json:"expires_at"`                          // 专线独立到期时间，nil 表示跟随订阅
	DedicatedOnly bool       `gorm:"default:false" json:"dedicated_only"` // 是否只显示专线节点
	LimitDevices  bool       `gorm:"default:false" json:"limit_devices"`  // 是否限制设备数量（false=不限制）
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserCustomNode) TableName() string {
	return "user_custom_nodes"
}

// CustomNodeSource 专线节点的「订阅来源」。
//
// 为什么需要：专线节点此前只能一次性导入（手工链接或订阅），订阅内容变了不会跟着更新，
// 时间一长节点全部失效；而且后台看不到自己当初导入的是哪个订阅链接，
// 想更新/更换/删除都无从下手。把来源单独建表后：
//
//	· 后台能看到订阅链接、节点数、上次同步时间与结果
//	· 可随时改链接、立即更新、删除整条来源（连同它导入的节点）
//	· 调度器按间隔自动重新拉取，订阅一变节点就跟着变
type CustomNodeSource struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"type:varchar(100)" json:"name"`
	// URL 订阅地址（唯一）
	URL string `gorm:"type:text;uniqueIndex" json:"url"`
	// Enabled 是否参与自动同步。
	// 不用 gorm default 标签：GORM 在 Create 时会把零值字段交给数据库默认值，
	// 那样「新建时就关掉自动同步」（false）会被写成 true。
	Enabled bool `gorm:"index" json:"enabled"`
	// IntervalHours 自动同步间隔（小时），0 表示只用「立即更新」手动同步。
	// 同样不用 default 标签，否则 0 会被默认值顶成 6，管理员设的「只手动更新」失效。
	IntervalHours int `json:"interval_hours"`
	// LastSyncAt 上次同步时间
	LastSyncAt *time.Time `json:"last_sync_at"`
	// LastStatus 上次同步结果：ok / error
	LastStatus string `gorm:"type:varchar(20)" json:"last_status"`
	// LastMessage 上次同步详情（新增/更新/停用数量或错误原因）
	LastMessage string `gorm:"type:text" json:"last_message"`
	// NodeTotal 上次同步时该来源的节点数
	NodeTotal int       `gorm:"default:0" json:"node_total"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (CustomNodeSource) TableName() string {
	return "custom_node_sources"
}
