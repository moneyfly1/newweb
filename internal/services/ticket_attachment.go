package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard/v2/internal/config"
)

// 工单附件存储服务：文件保存在 uploads/tickets/YYYY/MM/ 下，
// 元数据记录在 ticket_attachments 表（由 handler 写入）。

// ticketUploadMaxSize 单文件最大 20MB（图片/视频/附件综合）
const ticketUploadMaxSize = 20 * 1024 * 1024

// TicketUploadDir 附件根目录（位于 uploads 下）
const TicketUploadDir = "tickets"

// AllowedTicketExts 允许的附件扩展名（图片/视频/通用文档）
var AllowedTicketExts = map[string]bool{
	// 图片（heic/heif 兼容 iPhone 相册）
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true, ".svg": true,
	".heic": true, ".heif": true,
	// 视频
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true, ".m4v": true,
	// 文档/压缩
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".txt": true, ".log": true, ".zip": true, ".rar": true, ".7z": true, ".tar": true, ".gz": true,
	".csv": true, ".json": true, ".md": true,
}

// IsAllowedTicketFile 校验文件扩展名是否允许上传
func IsAllowedTicketFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return AllowedTicketExts[ext]
}

// ticketExtContentTypes 常见扩展名 → Content-Type（下载时设置，供浏览器正确打开）
func ticketExtContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	m := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif",
		".webp": "image/webp", ".svg": "image/svg+xml", ".heic": "image/heic", ".heif": "image/heif",
		".mp4": "video/mp4", ".webm": "video/webm",
		".mov": "video/quicktime", ".pdf": "application/pdf", ".zip": "application/zip",
		".txt": "text/plain", ".csv": "text/csv", ".json": "application/json", ".md": "text/markdown",
	}
	if ct, ok := m[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// TicketContentType 根据文件路径返回 Content-Type
func TicketContentType(path string) string {
	return ticketExtContentType(path)
}

// SaveTicketAttachment 保存上传的工单附件，返回相对存储路径（tickets/YYYY/MM/<uuid>.<ext>）与文件名。
// 返回的 relPath 不含根目录，供 handler 拼接完整路径与记录元数据。
func SaveTicketAttachment(src io.Reader, originalName string) (relPath string, fileName string, err error) {
	ext := strings.ToLower(filepath.Ext(originalName))
	if !IsAllowedTicketFile(originalName) {
		return "", "", fmt.Errorf("不支持的文件类型: %s（允许图片/视频/pdf/zip等）", ext)
	}

	now := time.Now()
	dir := filepath.Join(config.AppConfig.UploadDir, TicketUploadDir,
		now.Format("2006"), now.Format("01"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("创建附件目录失败: %w", err)
	}

	// 安全文件名：时间戳+随机，避免路径注入与重名
	fileName = fmt.Sprintf("%s_%d%s", now.Format("20060102_150405"), time.Now().UnixNano()%1000000, ext)
	absPath := filepath.Join(dir, fileName)

	// 限制大小（边读边限，超限中止）
	limited := io.LimitReader(src, ticketUploadMaxSize+1)
	out, err := os.Create(absPath)
	if err != nil {
		return "", "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	n, err := io.Copy(out, limited)
	if err != nil {
		os.Remove(absPath)
		return "", "", fmt.Errorf("写入文件失败: %w", err)
	}
	if n > ticketUploadMaxSize {
		os.Remove(absPath)
		return "", "", fmt.Errorf("文件过大，最大允许 20MB")
	}

	// 相对路径（tickets/2026/08/xxx.ext），供下载拼接
	rel := filepath.ToSlash(filepath.Join(TicketUploadDir, now.Format("2006"), now.Format("01"), fileName))
	return rel, fileName, nil
}
