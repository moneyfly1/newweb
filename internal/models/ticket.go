package models

import "time"

const (
	TicketStatusPending    = "pending"
	TicketStatusProcessing = "processing"
	TicketStatusResolved   = "resolved"
	TicketStatusClosed     = "closed"
	TicketStatusCancelled  = "cancelled"

	TicketTypeTechnical = "technical"
	TicketTypeBilling   = "billing"
	TicketTypeAccount   = "account"
	TicketTypeOther     = "other"

	TicketPriorityLow    = "low"
	TicketPriorityNormal = "normal"
	TicketPriorityHigh   = "high"
	TicketPriorityUrgent = "urgent"
)

type Ticket struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	TicketNo      string     `gorm:"type:varchar(50);uniqueIndex" json:"ticket_no"`
	UserID        uint       `gorm:"index" json:"user_id"`
	Title         string     `gorm:"type:varchar(200)" json:"title"`
	Content       string     `gorm:"type:text" json:"content"`
	Type          string     `gorm:"type:varchar(20);default:'other'" json:"type"`
	Status        string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Priority      string     `gorm:"type:varchar(20);default:'normal'" json:"priority"`
	AssignedTo    *int64     `gorm:"index" json:"assigned_to"`
	AdminNotes    *string    `gorm:"type:text" json:"admin_notes"`
	Rating        *int64     `json:"rating"`
	RatingComment *string    `gorm:"type:text" json:"rating_comment"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	ClosedAt      *time.Time `json:"closed_at"`
}

func (Ticket) TableName() string {
	return "tickets"
}

type TicketReply struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	TicketID  uint       `gorm:"index" json:"ticket_id"`
	UserID    uint       `gorm:"index" json:"user_id"`
	Content   string     `gorm:"type:text" json:"content"`
	IsAdmin   bool       `gorm:"default:false" json:"is_admin"`
	IsRead    bool       `gorm:"default:false" json:"is_read"`
	ReadBy    *uint      `gorm:"index" json:"read_by"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (TicketReply) TableName() string {
	return "ticket_replies"
}

type TicketAttachment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TicketID   uint      `gorm:"index" json:"ticket_id"`
	ReplyID    *int64    `gorm:"index" json:"reply_id"`
	FileName   string    `gorm:"type:varchar(255)" json:"file_name"`
	FilePath   string    `gorm:"type:varchar(500)" json:"file_path"`
	FileSize   *int64    `json:"file_size"`
	FileType   *string   `gorm:"type:varchar(50)" json:"file_type"`
	UploadedBy uint      `gorm:"index" json:"uploaded_by"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (TicketAttachment) TableName() string {
	return "ticket_attachments"
}

type TicketRead struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TicketID  uint      `gorm:"index" json:"ticket_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	ReadAt    time.Time `gorm:"autoCreateTime;autoUpdateTime" json:"read_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (TicketRead) TableName() string {
	return "ticket_reads"
}
