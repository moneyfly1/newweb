package handlers

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cboard/v2/internal/config"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// =============================================================
// 工单附件：先上传为 pending（ticket_id=0），创建工单/回复时携带
// attachment_ids 由后端绑定到对应 ticket/reply。下载/预览需鉴权
// （工单所有者或管理员），文件存于 uploads/tickets/YYYY/MM/。
// =============================================================

// ticketCurrentUser 获取当前用户 ID 与管理员标记
func ticketCurrentUser(c *gin.Context) (uint, bool) {
	uid := c.GetUint("user_id")
	isAdmin := false
	if u, ok := c.Get("user"); ok {
		if user, ok := u.(*models.User); ok && user.IsAdmin {
			isAdmin = true
		}
	}
	return uid, isAdmin
}

// ticketAttachmentURL 附件访问相对路径（前端 axios baseURL=/api/v1）
func ticketAttachmentURL(ticketID, attID uint) string {
	return "/tickets/" + strconv.FormatUint(uint64(ticketID), 10) +
		"/attachments/" + strconv.FormatUint(uint64(attID), 10)
}

// UploadTicketAttachment 上传工单附件（multipart 字段 files，可多文件）。
// 附件先落为 pending（ticket_id=0，uploaded_by=当前用户），随后由
// 创建工单/回复请求携带 attachment_ids 完成绑定；24h 未绑定的孤儿自动清理。
func UploadTicketAttachment(c *gin.Context) {
	uid, _ := ticketCurrentUser(c)
	db := database.GetDB()

	// 惰性清理本用户 24h 前未绑定的孤儿附件（记录+文件），避免磁盘/DB 膨胀
	cleanupPendingTicketAttachments(uid)

	form, err := c.MultipartForm()
	if err != nil {
		utils.BadRequest(c, "请选择要上传的文件")
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		utils.BadRequest(c, "请选择要上传的文件")
		return
	}
	if len(files) > 5 {
		utils.BadRequest(c, "单次最多上传 5 个文件")
		return
	}

	type fileResult struct {
		ID       uint   `json:"id"`
		FileName string `json:"file_name"`
		FileType string `json:"file_type"`
		FileSize int64  `json:"file_size"`
	}
	results := make([]fileResult, 0, len(files))

	for _, fh := range files {
		if !services.IsAllowedTicketFile(fh.Filename) {
			utils.BadRequest(c, "不支持的文件类型: "+fh.Filename+"（支持图片/视频/pdf/zip等）")
			return
		}
		src, err := fh.Open()
		if err != nil {
			utils.InternalError(c, "读取文件失败")
			return
		}

		relPath, _, err := services.SaveTicketAttachment(src, fh.Filename)
		src.Close()
		if err != nil {
			utils.BadRequest(c, err.Error())
			return
		}

		ftype := services.TicketContentType(fh.Filename)
		rec := models.TicketAttachment{
			TicketID:   0, // pending，等待创建工单/回复时绑定
			FileName:   fh.Filename,
			FilePath:   relPath,
			FileSize:   &fh.Size,
			FileType:   &ftype,
			UploadedBy: uid,
		}
		if err := db.Create(&rec).Error; err != nil {
			utils.InternalError(c, "保存附件记录失败")
			return
		}
		results = append(results, fileResult{ID: rec.ID, FileName: fh.Filename, FileType: ftype, FileSize: fh.Size})
	}

	utils.Success(c, gin.H{"files": results})
}

// DeleteTicketAttachment 删除尚未绑定的附件（仅限本人、pending 状态），
// 同步删除磁盘文件。已绑定到工单的附件不可删除。
func DeleteTicketAttachment(c *gin.Context) {
	uid, _ := ticketCurrentUser(c)
	attID, err := strconv.ParseUint(c.Param("attId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的附件ID")
		return
	}
	db := database.GetDB()
	var att models.TicketAttachment
	if err := db.Where("id = ? AND uploaded_by = ?", attID, uid).First(&att).Error; err != nil {
		utils.NotFound(c, "附件不存在")
		return
	}
	if att.TicketID != 0 {
		utils.BadRequest(c, "附件已绑定工单，无法删除")
		return
	}
	// 删除磁盘文件（忽略不存在错误）
	if att.FilePath != "" {
		full := filepath.Join(config.AppConfig.UploadDir, filepath.FromSlash(att.FilePath))
		if safePathUnderUploads(full) {
			os.Remove(full)
		}
	}
	if err := db.Delete(&att).Error; err != nil {
		utils.InternalError(c, "删除附件失败")
		return
	}
	utils.SuccessMessage(c, "附件已删除")
}

// DownloadTicketAttachment 下载/预览工单附件。鉴权：工单所有者或管理员。
// 图片/视频/pdf 内联预览；其余强制下载。
func DownloadTicketAttachment(c *gin.Context) {
	uid, isAdmin := ticketCurrentUser(c)
	ticketID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的工单ID")
		return
	}
	attID, err := strconv.ParseUint(c.Param("attId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的附件ID")
		return
	}
	db := database.GetDB()

	var ticket models.Ticket
	if err := db.First(&ticket, ticketID).Error; err != nil {
		utils.NotFound(c, "工单不存在")
		return
	}
	if !isAdmin && ticket.UserID != uid {
		utils.Forbidden(c, "无权访问此工单")
		return
	}

	var att models.TicketAttachment
	if err := db.Where("id = ? AND ticket_id = ?", attID, ticketID).First(&att).Error; err != nil {
		utils.NotFound(c, "附件不存在")
		return
	}

	fullPath := filepath.Join(config.AppConfig.UploadDir, filepath.FromSlash(att.FilePath))
	if !safePathUnderUploads(fullPath) {
		utils.NotFound(c, "附件路径异常")
		return
	}
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		utils.NotFound(c, "附件文件不存在")
		return
	}

	c.Header("Content-Type", services.TicketContentType(att.FilePath))
	c.Header("X-Content-Type-Options", "nosniff")
	if isInlinePreviewable(att.FilePath) {
		// 浏览器内联预览（图片/视频/pdf）
		c.File(fullPath)
	} else {
		// 下载
		disposition := "attachment"
		if !isASCIIFileName(att.FileName) {
			// 中文文件名用 RFC 5987 编码，避免乱码
			disposition = "attachment; filename*=UTF-8''" + urlPathEscape(att.FileName)
		} else {
			disposition = `attachment; filename="` + strings.ReplaceAll(att.FileName, `"`, "") + `"`
		}
		c.Header("Content-Disposition", disposition)
		c.File(fullPath)
	}
}

// BindTicketAttachments 创建工单/回复后把 pending 附件绑定到 ticket（reply 可选）。
// 只绑定当前用户上传、尚未绑定的附件；数量不匹配不报错（部分成功）。
func BindTicketAttachments(c *gin.Context, uid uint, ticketID uint, replyID *uint, ids []uint) {
	if len(ids) == 0 {
		return
	}
	db := database.GetDB()
	updates := map[string]interface{}{"ticket_id": ticketID}
	if replyID != nil {
		updates["reply_id"] = int64(*replyID)
	}
	db.Model(&models.TicketAttachment{}).
		Where("id IN ? AND uploaded_by = ? AND ticket_id = 0", ids, uid).
		Updates(updates)
}

// collectTicketAttachments 查询工单全部附件并生成访问路径
func collectTicketAttachments(ticketID uint) []gin.H {
	db := database.GetDB()
	var atts []models.TicketAttachment
	db.Where("ticket_id = ?", ticketID).Order("id ASC").Find(&atts)
	out := make([]gin.H, 0, len(atts))
	for _, a := range atts {
		item := gin.H{
			"id":         a.ID,
			"ticket_id":  a.TicketID,
			"reply_id":   a.ReplyID,
			"file_name":  a.FileName,
			"file_type":  a.FileType,
			"file_size":  a.FileSize,
			"created_at": a.CreatedAt,
			"url":        ticketAttachmentURL(ticketID, a.ID),
		}
		out = append(out, item)
	}
	return out
}

func cleanupPendingTicketAttachments(uid uint) {
	db := database.GetDB()
	var atts []models.TicketAttachment
	cutoff := time.Now().Add(-24 * time.Hour)
	db.Where("uploaded_by = ? AND ticket_id = 0 AND created_at < ?", uid, cutoff).Find(&atts)
	for _, a := range atts {
		if a.FilePath != "" {
			full := filepath.Join(config.AppConfig.UploadDir, filepath.FromSlash(a.FilePath))
			if safePathUnderUploads(full) {
				os.Remove(full)
			}
		}
		db.Delete(&models.TicketAttachment{}, a.ID)
	}
}

func safePathUnderUploads(fullPath string) bool {
	root := filepath.Clean(config.AppConfig.UploadDir)
	clean := filepath.Clean(fullPath)
	return strings.HasPrefix(clean, root+string(os.PathSeparator))
}

func isInlinePreviewable(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg",
		".mp4", ".webm", ".mov", ".m4v", ".pdf":
		return true
	}
	return false
}

func isASCIIFileName(name string) bool {
	for _, r := range name {
		if r > 127 {
			return false
		}
	}
	return true
}

func urlPathEscape(s string) string {
	// 简易百分号编码（UTF-8 逐字节）
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0x0F])
		}
	}
	return b.String()
}
