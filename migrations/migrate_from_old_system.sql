-- ============================================
-- 旧系统数据迁移脚本
-- 从 tempadmin 数据库迁移到新系统
-- ============================================

-- 注意事项：
-- 1. 旧系统密码有两种格式：bcrypt ($2y$10$...) 和 MD5 (32位hex)
-- 2. bcrypt 密码可以直接迁移，Go 的 bcrypt 兼容 PHP 的 $2y$ 格式
-- 3. MD5 密码无法直接迁移，需要用户重置密码
-- 4. 旧系统使用 QQ号 作为用户名，没有邮箱字段
-- 5. 需要为每个用户生成临时邮箱：username@migrate.local

-- ============================================
-- 第一步：迁移用户数据
-- ============================================

-- 创建临时表用于数据转换
CREATE TEMPORARY TABLE temp_old_users AS
SELECT
    id as old_id,
    username,
    password,
    CASE
        WHEN status = 1 THEN TRUE
        ELSE FALSE
    END as is_active,
    FROM_UNIXTIME(regtime) as created_at,
    FROM_UNIXTIME(lasttime) as last_login,
    -- 判断密码类型
    CASE
        WHEN password LIKE '$2y$10$%' THEN 'bcrypt'
        WHEN LENGTH(password) = 32 AND password REGEXP '^[a-f0-9]{32}$' THEN 'md5'
        ELSE 'unknown'
    END as password_type
FROM yg_user
WHERE activation = 1;  -- 只迁移已激活的用户

-- 插入用户数据到新系统
-- 注意：MD5 密码的用户需要设置一个临时密码，提示用户重置
INSERT INTO users (
    username,
    email,
    password,
    is_active,
    is_verified,
    is_admin,
    balance,
    created_at,
    updated_at,
    last_login,
    notes
)
SELECT
    username,
    CONCAT(username, '@migrate.local') as email,  -- 临时邮箱
    CASE
        WHEN password_type = 'bcrypt' THEN password  -- bcrypt 直接迁移
        ELSE '$2y$10$MIGRATION_REQUIRED_PLEASE_RESET_PASSWORD'  -- MD5 用户需要重置
    END as password,
    is_active,
    TRUE as is_verified,  -- 旧系统已激活用户视为已验证
    FALSE as is_admin,
    0.00 as balance,
    created_at,
    NOW() as updated_at,
    last_login,
    CASE
        WHEN password_type = 'md5' THEN CONCAT('旧系统用户，原密码为MD5格式，需要重置密码。原用户ID: ', old_id)
        ELSE CONCAT('从旧系统迁移。原用户ID: ', old_id)
    END as notes
FROM temp_old_users
ON DUPLICATE KEY UPDATE
    notes = CONCAT(notes, ' | 重复迁移尝试: ', NOW());

-- 创建用户ID映射表（用于后续迁移订阅和订单）
CREATE TEMPORARY TABLE user_id_mapping AS
SELECT
    t.old_id,
    u.id as new_id,
    t.username
FROM temp_old_users t
JOIN users u ON u.username = t.username;

-- ============================================
-- 第二步：迁移订阅数据
-- ============================================

-- 注意：需要先查看旧系统的 yg_dingyue 表结构
-- 这里提供一个基本框架，需要根据实际表结构调整

-- 示例（需要根据实际 yg_dingyue 表结构调整）：
/*
INSERT INTO subscriptions (
    user_id,
    package_id,
    subscription_url,
    device_limit,
    current_devices,
    is_active,
    status,
    expire_time,
    created_at,
    updated_at
)
SELECT
    m.new_id as user_id,
    NULL as package_id,  -- 旧系统可能没有套餐ID，需要手动映射
    d.subscription_url,  -- 需要确认字段名
    d.device_limit,
    d.current_devices,
    d.is_active,
    'active' as status,
    FROM_UNIXTIME(d.expire_time) as expire_time,
    FROM_UNIXTIME(d.created_at) as created_at,
    NOW() as updated_at
FROM yg_dingyue d
JOIN user_id_mapping m ON m.old_id = d.user_id
WHERE d.is_active = 1;
*/

-- ============================================
-- 第三步：迁移订单数据
-- ============================================

-- 示例（需要根据实际 yg_order 表结构调整）：
/*
INSERT INTO orders (
    order_no,
    user_id,
    package_id,
    amount,
    status,
    payment_method_name,
    payment_time,
    created_at,
    updated_at
)
SELECT
    o.order_no,
    m.new_id as user_id,
    NULL as package_id,  -- 需要手动映射
    o.amount,
    CASE
        WHEN o.status = 1 THEN 'paid'
        WHEN o.status = 0 THEN 'pending'
        WHEN o.status = 2 THEN 'cancelled'
        ELSE 'pending'
    END as status,
    o.payment_method as payment_method_name,
    FROM_UNIXTIME(o.payment_time) as payment_time,
    FROM_UNIXTIME(o.created_at) as created_at,
    NOW() as updated_at
FROM yg_order o
JOIN user_id_mapping m ON m.old_id = o.user_id;
*/

-- ============================================
-- 第四步：迁移设备日志（可选）
-- ============================================

-- 注意：yg_device_log 表结构与新系统的 devices 表可能不完全匹配
-- 需要根据实际需求决定是否迁移

-- ============================================
-- 验证和清理
-- ============================================

-- 查看迁移统计
SELECT
    '用户迁移统计' as category,
    COUNT(*) as total_count,
    SUM(CASE WHEN notes LIKE '%MD5%' THEN 1 ELSE 0 END) as md5_password_count,
    SUM(CASE WHEN notes NOT LIKE '%MD5%' THEN 1 ELSE 0 END) as bcrypt_password_count
FROM users
WHERE notes LIKE '%从旧系统迁移%';

-- 列出需要重置密码的用户
SELECT
    id,
    username,
    email,
    notes
FROM users
WHERE notes LIKE '%MD5%'
ORDER BY id;

-- 清理临时表
DROP TEMPORARY TABLE IF EXISTS temp_old_users;
DROP TEMPORARY TABLE IF EXISTS user_id_mapping;

-- ============================================
-- 迁移后操作建议
-- ============================================

-- 1. 通知所有 MD5 密码用户重置密码
-- 2. 要求用户更新邮箱地址（从 @migrate.local 改为真实邮箱）
-- 3. 验证订阅和订单数据的完整性
-- 4. 检查设备限制是否正确
-- 5. 备份旧数据库以防需要回滚
