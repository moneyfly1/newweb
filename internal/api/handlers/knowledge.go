package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
)

// ==================== 知识库（对齐 myweb knowledge）====================

// GetKnowledgeCategories 用户端：分类列表。
func GetKnowledgeCategories(c *gin.Context) {
	var cats []models.KnowledgeCategory
	database.GetDB().Where("is_active = ?", true).Order("sort_order ASC, id ASC").Find(&cats)
	utils.Success(c, cats)
}

// GetKnowledgeArticles 用户端：文章列表（可按分类过滤）。
func GetKnowledgeArticles(c *gin.Context) {
	db := database.GetDB().Model(&models.KnowledgeArticle{}).Where("is_active = ?", true)
	if catID := c.Query("category_id"); catID != "" {
		db = db.Where("category_id = ?", catID)
	}
	if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
		db = db.Where("title LIKE ? OR content LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	var articles []models.KnowledgeArticle
	db.Order("sort_order ASC, id DESC").Limit(200).Find(&articles)
	utils.Success(c, articles)
}

// GetKnowledgeArticle 用户端：文章详情（浏览量 +1）。
func GetKnowledgeArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的文章ID")
		return
	}
	db := database.GetDB()
	var article models.KnowledgeArticle
	if err := db.Where("id = ? AND is_active = ?", id, true).First(&article).Error; err != nil {
		utils.NotFound(c, "文章不存在")
		return
	}
	db.Model(&models.KnowledgeArticle{}).Where("id = ?", article.ID).
		UpdateColumn("view_count", models.KnowledgeArticle{}.ViewCount+article.ViewCount+1)
	utils.Success(c, article)
}

// AdminListKnowledgeCategories 管理端：分类列表（含停用）。
func AdminListKnowledgeCategories(c *gin.Context) {
	var cats []models.KnowledgeCategory
	database.GetDB().Order("sort_order ASC, id ASC").Find(&cats)
	utils.Success(c, cats)
}

// AdminCreateKnowledgeCategory 管理端：创建分类。
func AdminCreateKnowledgeCategory(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Icon      string `json:"icon"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	cat := models.KnowledgeCategory{Name: req.Name, Icon: req.Icon, SortOrder: req.SortOrder, IsActive: true}
	if err := database.GetDB().Create(&cat).Error; err != nil {
		utils.InternalError(c, "创建分类失败")
		return
	}
	utils.Success(c, cat)
}

// AdminUpdateKnowledgeCategory 管理端：更新分类。
func AdminUpdateKnowledgeCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的分类ID")
		return
	}
	var req struct {
		Name      string `json:"name"`
		Icon      string `json:"icon"`
		SortOrder *int   `json:"sort_order"`
		IsActive  *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if len(updates) > 0 {
		if err := database.GetDB().Model(&models.KnowledgeCategory{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			utils.InternalError(c, "更新分类失败")
			return
		}
	}
	utils.SuccessMessage(c, "更新成功")
}

// AdminDeleteKnowledgeCategory 管理端：删除分类（同时删除其文章）。
func AdminDeleteKnowledgeCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的分类ID")
		return
	}
	db := database.GetDB()
	db.Where("category_id = ?", id).Delete(&models.KnowledgeArticle{})
	if err := db.Delete(&models.KnowledgeCategory{}, id).Error; err != nil {
		utils.InternalError(c, "删除分类失败")
		return
	}
	utils.SuccessMessage(c, "已删除")
}

// AdminListKnowledgeArticles 管理端：文章列表。
func AdminListKnowledgeArticles(c *gin.Context) {
	var articles []models.KnowledgeArticle
	database.GetDB().Order("id DESC").Limit(500).Find(&articles)
	utils.Success(c, articles)
}

// AdminCreateKnowledgeArticle 管理端：创建文章。
func AdminCreateKnowledgeArticle(c *gin.Context) {
	var req struct {
		CategoryID uint   `json:"category_id" binding:"required"`
		Title      string `json:"title" binding:"required"`
		Content    string `json:"content" binding:"required"`
		Summary    string `json:"summary"`
		SortOrder  int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	article := models.KnowledgeArticle{
		CategoryID: req.CategoryID, Title: req.Title, Content: req.Content,
		Summary: req.Summary, SortOrder: req.SortOrder, IsActive: true,
	}
	if err := database.GetDB().Create(&article).Error; err != nil {
		utils.InternalError(c, "创建文章失败")
		return
	}
	utils.CreateAuditLog(c, "create_knowledge", "knowledge", article.ID, fmt.Sprintf("创建知识库文章: %s", article.Title))
	utils.Success(c, article)
}

// AdminUpdateKnowledgeArticle 管理端：更新文章。
func AdminUpdateKnowledgeArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的文章ID")
		return
	}
	var req struct {
		CategoryID *uint  `json:"category_id"`
		Title      string `json:"title"`
		Content    string `json:"content"`
		Summary    string `json:"summary"`
		SortOrder  *int   `json:"sort_order"`
		IsActive   *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	updates := map[string]interface{}{}
	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Summary != "" {
		updates["summary"] = req.Summary
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if len(updates) > 0 {
		if err := database.GetDB().Model(&models.KnowledgeArticle{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			utils.InternalError(c, "更新文章失败")
			return
		}
	}
	utils.SuccessMessage(c, "更新成功")
}

// AdminDeleteKnowledgeArticle 管理端：删除文章。
func AdminDeleteKnowledgeArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的文章ID")
		return
	}
	if err := database.GetDB().Delete(&models.KnowledgeArticle{}, id).Error; err != nil {
		utils.InternalError(c, "删除文章失败")
		return
	}
	utils.SuccessMessage(c, "已删除")
}
