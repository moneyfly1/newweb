// Package userlevel 用户等级的公共逻辑：匹配、同步、折扣与"下一等级"。
//
// 为什么单独成包（对齐 myweb 2026-10-28 的收敛）：
// 等级逻辑此前散在订单/折扣/设备升级等多处，各自实现、都漏了 is_active 判断，
// 导致"停用等级仍能打折"。现在统一在这里，调用方只拿结果。
package userlevel

import (
	"cboard/v2/internal/models"

	"gorm.io/gorm"
)

// ActiveLevels 启用中的等级，按 level_order 升序。
func ActiveLevels(db *gorm.DB) ([]models.UserLevel, error) {
	var levels []models.UserLevel
	err := db.Where("is_active = ?", true).Order("level_order ASC").Find(&levels).Error
	return levels, err
}

// Match 返回累计消费命中的最高达标等级；没有达标等级时返回 nil。
func Match(db *gorm.DB, totalConsumption float64) (*models.UserLevel, error) {
	levels, err := ActiveLevels(db)
	if err != nil {
		return nil, err
	}
	var target *models.UserLevel
	for i := range levels {
		if totalConsumption >= levels[i].MinConsumption &&
			(target == nil || levels[i].LevelOrder > target.LevelOrder) {
			target = &levels[i]
		}
	}
	return target, nil
}

// Plan 只计算同步结果、不写库：返回目标等级、是否需要更新。
// 只升不降：当前等级的 level_order 高于目标时保持原样。
func Plan(db *gorm.DB, user *models.User) (*models.UserLevel, bool, error) {
	target, err := Match(db, user.TotalConsumption)
	if err != nil || target == nil {
		return nil, false, err
	}
	if user.UserLevelID == nil {
		return target, true, nil
	}
	var current models.UserLevel
	if err := db.First(&current, *user.UserLevelID).Error; err == nil {
		if current.LevelOrder > target.LevelOrder {
			return &current, false, nil // 只升不降
		}
		return target, current.ID != target.ID, nil
	}
	// 当前等级已被删除：重新挂上目标等级
	return target, true, nil
}

// Sync 按累计消费同步用户等级（只升不降），返回同步后的等级与是否变更。
// 用户没有命中任何等级时保持原样（不降级）。
func Sync(tx *gorm.DB, user *models.User) (*models.UserLevel, bool, error) {
	target, changed, err := Plan(tx, user)
	if err != nil || target == nil || !changed {
		return target, false, err
	}
	levelID := target.ID
	user.UserLevelID = &levelID
	// 只更新等级列：调用方手里的 user 可能只 Select 了少数字段，
	// 整行 Save 会把未加载的列写成零值。
	if err := tx.Model(&models.User{}).Where("id = ?", user.ID).
		Update("user_level_id", levelID).Error; err != nil {
		return nil, false, err
	}
	return target, true, nil
}

// ForUser 取用户当前**生效**的等级：等级被删除或停用时返回 nil。
// 折扣计算与前台展示都必须走这里，保证"停用即不生效"。
func ForUser(db *gorm.DB, userID uint) (*models.UserLevel, error) {
	var user models.User
	if err := db.Select("id", "user_level_id").First(&user, userID).Error; err != nil {
		return nil, err
	}
	if user.UserLevelID == nil {
		return nil, nil
	}
	var level models.UserLevel
	if err := db.Where("id = ? AND is_active = ?", *user.UserLevelID, true).First(&level).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &level, nil
}
