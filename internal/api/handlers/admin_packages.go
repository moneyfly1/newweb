package handlers

import (
	"fmt"
	"strconv"
	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"
	"github.com/gin-gonic/gin"
)

func AdminListPackages(c *gin.Context) {
	db := database.GetDB()
	p := utils.GetPagination(c)

	query := db.Model(&models.Package{})
	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	var total int64
	query.Count(&total)

	var packages []models.Package
	query.Order(p.OrderClause()).Offset(p.Offset()).Limit(p.PageSize).Find(&packages)

	utils.SuccessPage(c, packages, total, p.Page, p.PageSize)
}

func AdminGetPackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的套餐ID")
		return
	}
	var pkg models.Package
	if err := database.GetDB().First(&pkg, id).Error; err != nil {
		utils.NotFound(c, "套餐不存在")
		return
	}
	utils.Success(c, pkg)
}

func AdminCreatePackage(c *gin.Context) {
	var pkg models.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := database.GetDB().Create(&pkg).Error; err != nil {
		utils.InternalError(c, "创建套餐失败")
		return
	}
	utils.CreateAuditLog(c, "create_package", "package", pkg.ID, fmt.Sprintf("创建套餐: %s", pkg.Name))
	utils.Success(c, pkg)
}

func AdminUpdatePackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的套餐ID")
		return
	}
	db := database.GetDB()
	var pkg models.Package
	if err := db.First(&pkg, id).Error; err != nil {
		utils.NotFound(c, "套餐不存在")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	allowed := map[string]bool{
		"name": true, "description": true, "price": true, "duration_days": true,
		"device_limit": true, "is_active": true, "sort_order": true, "features": true,
		"original_price": true, "discount_text": true, "badge": true,
	}
	updates := make(map[string]interface{})
	for k, v := range req {
		if allowed[k] {
			updates[k] = v
		}
	}
	if len(updates) == 0 {
		utils.BadRequest(c, "无有效更新字段")
		return
	}
	if err := db.Model(&pkg).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新套餐失败")
		return
	}
	utils.CreateAuditLog(c, "update_package", "package", uint(id), fmt.Sprintf("更新套餐: %s", pkg.Name))
	utils.Success(c, pkg)
}

func AdminDeletePackage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的套餐ID")
		return
	}
	if err := database.GetDB().Delete(&models.Package{}, id).Error; err != nil {
		utils.InternalError(c, "删除套餐失败")
		return
	}
	utils.CreateAuditLog(c, "delete_package", "package", uint(id), "删除套餐")
	utils.InvalidatePublicCache("public_packages")
	utils.SuccessMessage(c, "套餐已删除")
}

// ==================== Node Management ====================

