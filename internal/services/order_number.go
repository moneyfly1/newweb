package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	OrderNoPrefixOrder    = "ORD"
	OrderNoPrefixRecharge = "RCH"
	OrderNoPrefixUpgrade  = "UPG"
)

// GenerateBusinessOrderNo creates a readable daily sequence number such as ORD202607020001.
func GenerateBusinessOrderNo(db *gorm.DB, prefix string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("数据库连接为空")
	}
	base := prefix + time.Now().Format("20060102")
	maxSeq := 0

	for _, table := range []string{"orders", "recharge_records"} {
		var orderNos []string
		if err := db.Table(table).
			Select("order_no").
			Where("order_no LIKE ?", base+"%").
			Find(&orderNos).Error; err != nil {
			return "", err
		}
		for _, orderNo := range orderNos {
			if !strings.HasPrefix(orderNo, base) {
				continue
			}
			seq, err := strconv.Atoi(orderNo[len(base):])
			if err == nil && seq > maxSeq {
				maxSeq = seq
			}
		}
	}

	for seq := maxSeq + 1; seq <= maxSeq+1000; seq++ {
		candidate := fmt.Sprintf("%s%04d", base, seq)
		exists, err := businessOrderNoExists(db, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("生成订单号失败，请稍后重试")
}

func businessOrderNoExists(db *gorm.DB, orderNo string) (bool, error) {
	var count int64
	if err := db.Table("orders").Where("order_no = ?", orderNo).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if err := db.Table("recharge_records").Where("order_no = ?", orderNo).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
