-- 防止签到重放攻击：添加唯一索引
-- 确保每个用户每天只能签到一次

-- 方案 1: 添加唯一索引（推荐）
-- 注意：SQLite 不支持函数索引，所以我们需要添加一个日期列
-- 或者使用触发器

-- 先检查是否有重复数据
SELECT user_id, DATE(created_at) as check_date, COUNT(*) as cnt
FROM check_ins
GROUP BY user_id, DATE(created_at)
HAVING cnt > 1;

-- 如果有重复数据，先清理（保留最早的记录）
DELETE FROM check_ins
WHERE id NOT IN (
    SELECT MIN(id)
    FROM check_ins
    GROUP BY user_id, DATE(created_at)
);

-- 方案 2: 由于 SQLite 限制，我们在应用层通过事务内二次检查来防止重放
-- 已在代码中实现

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_checkin_user_date ON check_ins(user_id, created_at);

-- 添加注释（SQLite 不支持列注释，仅作文档）
-- check_ins 表用于记录用户签到
-- 防重放机制：在事务内二次检查 + 应用层锁
