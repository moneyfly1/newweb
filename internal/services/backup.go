package services

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/services/git"
	"cboard/v2/internal/utils"
)

type BackupResult struct {
	DBPath   string
	ZipPath  string
	Filename string
	Size     int64
	ZipSize  int64
}

// PerformBackup creates a database backup organized in backups/YYYY/MM/ folders.
func PerformBackup() (*BackupResult, error) {
	now := time.Now()
	backupDir := filepath.Join("backups", now.Format("2006"), now.Format("01"))
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %s", err.Error())
	}

	srcPath := "cboard.db"
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("数据库文件不存在，仅支持 SQLite 备份")
	}

	timestamp := now.Format("20060102_150405")
	dbBackupPath := filepath.Join(backupDir, fmt.Sprintf("cboard_backup_%s.db", timestamp))

	// #nosec G304 -- srcPath is a fixed constant.
	src, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %s", err.Error())
	}
	defer src.Close()

	// #nosec G304 -- dbBackupPath is server-generated under fixed backups directory.
	dst, err := os.Create(dbBackupPath)
	if err != nil {
		return nil, fmt.Errorf("创建备份文件失败: %s", err.Error())
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("备份失败: %s", err.Error())
	}

	zipPath := filepath.Join(backupDir, fmt.Sprintf("cboard_backup_%s.zip", timestamp))
	// #nosec G304 -- zipPath is server-generated under fixed backups directory.
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return nil, fmt.Errorf("创建ZIP文件失败: %s", err.Error())
	}
	zipWriter := zip.NewWriter(zipFile)

	if err := backupAddFileToZip(zipWriter, dbBackupPath, filepath.Base(dbBackupPath)); err != nil {
		zipWriter.Close()
		zipFile.Close()
		return nil, fmt.Errorf("添加数据库到ZIP失败: %s", err.Error())
	}

	if err := zipWriter.Close(); err != nil {
		zipFile.Close()
		return nil, fmt.Errorf("关闭ZIP写入失败: %s", err.Error())
	}
	if err := zipFile.Close(); err != nil {
		return nil, fmt.Errorf("关闭ZIP文件失败: %s", err.Error())
	}

	dbInfo, _ := os.Stat(dbBackupPath)
	zipInfo, _ := os.Stat(zipPath)

	return &BackupResult{
		DBPath:   dbBackupPath,
		ZipPath:  zipPath,
		Filename: filepath.Base(dbBackupPath),
		Size:     dbInfo.Size(),
		ZipSize:  zipInfo.Size(),
	}, nil
}

// UploadBackupToGitHub uploads a backup zip to the configured GitHub repo.
func UploadBackupToGitHub(zipPath string) {
	settings := utils.GetSettings("backup_github_enabled", "backup_github_token", "backup_github_repo")
	if settings["backup_github_enabled"] != "true" && settings["backup_github_enabled"] != "1" {
		return
	}
	token := settings["backup_github_token"]
	repo := settings["backup_github_repo"]
	if token == "" || repo == "" {
		return
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return
	}

	client := git.NewClient(git.PlatformGitHub, token, parts[0], parts[1])
	err := client.UploadBackupWithProgress(zipPath, func(progress int, message string) {
		log.Printf("[AutoBackup] GitHub上传进度: %d%% - %s", progress, message)
	})
	if err != nil {
		log.Printf("[AutoBackup] GitHub上传失败: %v", err)
		utils.SysError("backup", "GitHub自动备份上传失败", err.Error())
	} else {
		log.Printf("[AutoBackup] GitHub上传成功: %s", filepath.Base(zipPath))
		utils.SysInfo("backup", fmt.Sprintf("GitHub自动备份上传成功: %s", filepath.Base(zipPath)))
	}
}

func backupAddFileToZip(zipWriter *zip.Writer, filePath, nameInZip string) error {
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

// RestoreBackup restores the database from a backup file. It creates a safety
// backup first, then swaps the database file and reopens the connection.
func RestoreBackup(backupRelPath string) (safetyBackupName string, err error) {
	clean := filepath.Clean(backupRelPath)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("非法路径")
	}
	if !strings.HasSuffix(clean, ".db") {
		return "", fmt.Errorf("仅支持恢复 .db 备份文件")
	}

	backupFullPath := filepath.Join("backups", clean)
	if _, err := os.Stat(backupFullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("备份文件不存在: %s", clean)
	}

	log.Printf("[Restore] 开始恢复数据库, 备份文件: %s", clean)
	utils.SysInfo("restore", fmt.Sprintf("开始恢复数据库: %s", clean))

	safety, err := PerformBackup()
	if err != nil {
		return "", fmt.Errorf("创建恢复前安全备份失败: %w", err)
	}
	log.Printf("[Restore] 已创建安全备份: %s", safety.Filename)

	dbPath := "cboard.db"

	if err := database.CheckpointWAL(); err != nil {
		log.Printf("[Restore] WAL checkpoint 警告: %v", err)
	}

	database.Close()

	os.Remove(dbPath + "-wal")
	os.Remove(dbPath + "-shm")

	if err := copyFile(backupFullPath, dbPath); err != nil {
		log.Printf("[Restore] 文件替换失败, 尝试回滚: %v", err)
		_ = copyFile(safety.DBPath, dbPath)
		_ = database.ReopenDB()
		return "", fmt.Errorf("恢复失败: %w", err)
	}

	if err := database.ReopenDB(); err != nil {
		log.Printf("[Restore] 重新打开数据库失败, 尝试回滚: %v", err)
		_ = copyFile(safety.DBPath, dbPath)
		_ = database.ReopenDB()
		return "", fmt.Errorf("重新打开数据库失败: %w", err)
	}

	if err := database.AutoMigrate(); err != nil {
		log.Printf("[Restore] 数据库迁移警告: %v", err)
	}

	log.Printf("[Restore] 数据库恢复成功")
	utils.SysInfo("restore", fmt.Sprintf("数据库恢复成功, 安全备份: %s", safety.Filename))
	return safety.Filename, nil
}

func copyFile(src, dst string) error {
	// #nosec G304 -- src is validated by caller
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// #nosec G304 -- dst is a fixed known path
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
