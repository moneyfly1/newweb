package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"cboard/v2/internal/services"
	"cboard/v2/internal/services/git"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func AdminCreateBackup(c *gin.Context) {
	result, err := services.PerformBackup()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	zipPath := result.ZipPath
	zipInfo, _ := os.Stat(zipPath)

	response := gin.H{
		"filename":   result.Filename,
		"size":       result.Size,
		"created_at": time.Now(),
	}

	// Check if GitHub backup is enabled
	settings := utils.GetSettings("backup_github_enabled", "backup_github_token", "backup_github_repo")
	if settings["backup_github_enabled"] == "true" || settings["backup_github_enabled"] == "1" {
		token := settings["backup_github_token"]
		repo := settings["backup_github_repo"]

		if token != "" && repo != "" {
			// Parse owner/repo
			parts := strings.SplitN(repo, "/", 2)
			if len(parts) == 2 {
				owner := parts[0]
				repoName := parts[1]

				// Generate task ID
				taskID := uuid.New().String()

				// Create upload status
				statusManager := git.GetUploadStatusManager()
				status := &git.UploadStatus{
					Status:    "uploading",
					Progress:  0,
					Message:   "准备上传...",
					StartTime: time.Now(),
					FileName:  filepath.Base(zipPath),
					FileSize:  zipInfo.Size(),
				}
				statusManager.SetStatus(taskID, status)

				// Start async upload
				go func() {
					client := git.NewClient(git.PlatformGitHub, token, owner, repoName)
					err := client.UploadBackupWithProgress(zipPath, func(progress int, message string) {
						statusManager.UpdateStatus(taskID, "uploading", message, progress)
					})

					if err != nil {
						statusManager.UpdateError(taskID, err)
					} else {
						statusManager.UpdateStatus(taskID, "success", "上传成功", 100)
					}
				}()

				response["task_id"] = taskID
				response["github_upload"] = "started"
			}
		}
	}

	utils.CreateAuditLog(c, "create_backup", "backup", 0, fmt.Sprintf("创建备份: %s", result.Filename))
	utils.Success(c, response)
}

// addFileToZip adds a file to the zip archive
func addFileToZip(zipWriter *zip.Writer, filePath, nameInZip string) error {
	if strings.Contains(nameInZip, "..") || strings.ContainsAny(nameInZip, `/\`) {
		return fmt.Errorf("invalid zip entry name")
	}
	// #nosec G304 -- filePath comes from controlled backup generation flow.
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := zipWriter.Create(nameInZip)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

func AdminListBackups(c *gin.Context) {
	backupDir := "backups"
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		utils.Success(c, []interface{}{})
		return
	}

	type BackupInfo struct {
		Filename  string    `json:"filename"`
		Path      string    `json:"path"`
		Size      int64     `json:"size"`
		CreatedAt time.Time `json:"created_at"`
	}

	var backups []BackupInfo
	_ = filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".db") {
			return nil
		}
		relPath, _ := filepath.Rel(backupDir, path)
		backups = append(backups, BackupInfo{
			Filename:  info.Name(),
			Path:      relPath,
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
		return nil
	})

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	utils.Success(c, backups)
}

func AdminRestoreBackup(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请指定备份文件路径")
		return
	}

	safetyName, err := services.RestoreBackup(req.Path)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.CreateAuditLog(c, "restore_backup", "backup", 0, fmt.Sprintf("恢复数据库: %s", req.Path))
	utils.Success(c, gin.H{
		"message":       "数据库恢复成功",
		"safety_backup": safetyName,
	})
}

func AdminListGitHubBackups(c *gin.Context) {
	settings := utils.GetSettings("backup_github_enabled", "backup_github_token", "backup_github_repo")
	if settings["backup_github_enabled"] != "true" && settings["backup_github_enabled"] != "1" {
		utils.BadRequest(c, "GitHub 备份未启用")
		return
	}
	token := settings["backup_github_token"]
	repo := settings["backup_github_repo"]
	if token == "" || repo == "" {
		utils.BadRequest(c, "GitHub Token 或仓库地址未配置")
		return
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		utils.BadRequest(c, "仓库地址格式错误，应为 owner/repo")
		return
	}

	client := git.NewClient(git.PlatformGitHub, token, parts[0], parts[1])
	backups, err := client.ListBackups()
	if err != nil {
		utils.InternalError(c, "获取 GitHub 备份列表失败: "+err.Error())
		return
	}
	utils.Success(c, backups)
}

func AdminRestoreGitHubBackup(c *gin.Context) {
	var req struct {
		Path        string `json:"path" binding:"required"`
		DownloadURL string `json:"download_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	settings := utils.GetSettings("backup_github_token", "backup_github_repo")
	token := settings["backup_github_token"]
	repo := settings["backup_github_repo"]
	if token == "" || repo == "" {
		utils.BadRequest(c, "GitHub 未配置")
		return
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		utils.BadRequest(c, "仓库地址格式错误")
		return
	}

	client := git.NewClient(git.PlatformGitHub, token, parts[0], parts[1])

	localZip := filepath.Join("backups", "_github_download", filepath.Base(req.Path))
	if err := client.DownloadFile(req.DownloadURL, localZip); err != nil {
		utils.InternalError(c, "下载备份文件失败: "+err.Error())
		return
	}
	defer os.Remove(localZip)

	dbPath, err := extractDBFromZip(localZip)
	if err != nil {
		utils.InternalError(c, "解压备份文件失败: "+err.Error())
		return
	}

	relPath, _ := filepath.Rel("backups", dbPath)
	safetyName, err := services.RestoreBackup(relPath)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	os.Remove(dbPath)

	utils.CreateAuditLog(c, "restore_github_backup", "backup", 0, fmt.Sprintf("从 GitHub 恢复: %s", req.Path))
	utils.Success(c, gin.H{
		"message":       "从 GitHub 备份恢复成功",
		"safety_backup": safetyName,
	})
}

func extractDBFromZip(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".db") {
			continue
		}
		if strings.Contains(f.Name, "..") || strings.Contains(f.Name, "/") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		outPath := filepath.Join("backups", "_github_download", f.Name)
		if err := os.MkdirAll(filepath.Dir(outPath), 0750); err != nil {
			rc.Close()
			return "", err
		}
		out, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return "", err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return "", err
		}
		return outPath, nil
	}
	return "", fmt.Errorf("ZIP 中未找到 .db 文件")
}

func AdminGetUploadStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.BadRequest(c, "任务ID不能为空")
		return
	}
	statusManager := git.GetUploadStatusManager()
	status, exists := statusManager.GetStatus(taskID)
	if !exists {
		utils.NotFound(c, "未找到该上传任务")
		return
	}
	utils.Success(c, status)
}

func AdminTestGitHubConnection(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
		Repo  string `json:"repo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Fall back to saved settings
		settings := utils.GetSettings("backup_github_token", "backup_github_repo")
		req.Token = settings["backup_github_token"]
		req.Repo = settings["backup_github_repo"]
	}
	if req.Token == "" {
		utils.BadRequest(c, "Token不能为空")
		return
	}
	if req.Repo == "" {
		utils.BadRequest(c, "仓库地址不能为空")
		return
	}
	repo := strings.TrimSpace(req.Repo)
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.Trim(repo, "/")
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		utils.BadRequest(c, "仓库地址格式错误，应为 owner/repo")
		return
	}
	client := git.NewClient(git.PlatformGitHub, req.Token, parts[0], parts[1])
	if err := client.TestConnection(); err != nil {
		utils.BadRequest(c, "GitHub 连接测试失败: "+err.Error())
		return
	}
	utils.CreateAuditLog(c, "test_github", "backup", 0, "测试GitHub连接")
	utils.Success(c, gin.H{"message": "GitHub 连接测试成功"})
}

