package handlers

import (
	"fmt"
	"strings"

	"cboard/v2/internal/services"
	"cboard/v2/internal/services/git"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// ── GitHub 节点文件同步 (admin) ──

func AdminGithubNodesStatus(c *gin.Context) {
	svc := services.GetGithubNodesService()
	utils.Success(c, svc.Status())
}

// AdminGithubNodesTest 测试连接：优先使用请求体中的值（未保存的设置），否则回退到已保存设置。
func AdminGithubNodesTest(c *gin.Context) {
	var req struct {
		Token  string `json:"token"`
		Repo   string `json:"repo"`
		Branch string `json:"branch"`
		Path   string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Fall back to saved settings
		settings := utils.GetSettings("gh_nodes_token", "gh_nodes_repo", "gh_nodes_branch", "gh_nodes_path")
		req.Token = settings["gh_nodes_token"]
		req.Repo = settings["gh_nodes_repo"]
		req.Branch = settings["gh_nodes_branch"]
		req.Path = settings["gh_nodes_path"]
	}

	req.Token = strings.TrimSpace(req.Token)
	req.Repo = strings.TrimSpace(req.Repo)
	req.Branch = strings.TrimSpace(req.Branch)
	req.Path = strings.TrimSpace(req.Path)

	if req.Token == "" {
		utils.BadRequest(c, "GitHub Token 不能为空")
		return
	}
	if req.Repo == "" {
		req.Repo = "moneyfly006/nodes"
	}
	if req.Branch == "" {
		req.Branch = "main"
	}
	if req.Path == "" {
		req.Path = "nodes"
	}

	owner, repo, err := services.ParseGithubRepoAddress(req.Repo)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	client := git.NewClient(git.PlatformGitHub, req.Token, owner, repo)
	entries, err := client.ListDirWithRef(req.Path, req.Branch)
	if err != nil {
		utils.BadRequest(c, fmt.Sprintf("连接失败: %s（请检查 Token 权限、仓库地址、分支和目录是否正确）", err.Error()))
		return
	}

	fileCount, dirCount := 0, 0
	for _, e := range entries {
		if e.Type == "dir" {
			dirCount++
		} else if e.Type == "file" {
			fileCount++
		}
	}
	utils.Success(c, gin.H{
		"message":    fmt.Sprintf("连接成功：目录 %s 包含 %d 个文件、%d 个子目录", req.Path, fileCount, dirCount),
		"file_count": fileCount,
	})
}

func AdminGithubNodesSync(c *gin.Context) {
	svc := services.GetGithubNodesService()
	if err := svc.Start(); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.SuccessMessage(c, "同步任务已启动")
}

func AdminGithubNodesLogs(c *gin.Context) {
	svc := services.GetGithubNodesService()
	utils.Success(c, svc.GetLogs())
}

func AdminGithubNodesClearLogs(c *gin.Context) {
	svc := services.GetGithubNodesService()
	svc.ClearLogs()
	utils.SuccessMessage(c, "日志已清空")
}
