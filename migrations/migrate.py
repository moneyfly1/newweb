#!/usr/bin/env python3
"""迁移旧系统用户和订单数据到新系统SQLite"""

import re
import sqlite3
import hashlib
from datetime import datetime

DUMP_FILE = "/Users/apple/v2/tempadmin_2026-02-24_01-30-01_mysql_data.sql"
DB_FILE = "/Users/apple/v2/cboard.db"

def parse_users(sql):
    """从SQL dump提取yg_user数据"""
    pattern = r"INSERT INTO `yg_user` VALUES (.+?);"
    matches = re.findall(pattern, sql, re.DOTALL)
    users = []
    for match in matches:
        rows = re.findall(r"\((\d+),'([^']+)','([^']+)','(\d+)','(\d+)',(\d+),(\d+)\)", match)
        for row in rows:
            uid, username, password, regtime, lasttime, status, activation = row
            if activation == '1':  # 只迁移已激活用户
                users.append({
                    'old_id': int(uid),
                    'username': username,
                    'password': password,
                    'is_active': status == '1',
                    'created_at': datetime.fromtimestamp(int(regtime)).isoformat(),
                    'last_login': datetime.fromtimestamp(int(lasttime)).isoformat() if int(lasttime) > 0 else None,
                    'is_bcrypt': password.startswith('$2y$') or password.startswith('$2a$'),
                })
    return users

def parse_orders(sql):
    """从SQL dump提取yg_order数据"""
    pattern = r"INSERT INTO `yg_order` VALUES (.+?);"
    matches = re.findall(pattern, sql, re.DOTALL)
    orders = []
    for match in matches:
        # 解析每行订单
        row_pattern = r"\((\d+),'([^']+)',(\d+),'([^']+)',([\d.]+),(\d+),(\d+),(?:'([^']*)'|NULL),'([^']+)',(?:'([^']*)'|NULL),(?:'([^']*)'|NULL)\)"
        rows = re.findall(row_pattern, match)
        for row in rows:
            oid, user_name, plan_id, order_no, amount, days, status, pay_method, create_time, pay_time, pay_no = row
            orders.append({
                'order_no': order_no,
                'user_name': user_name,
                'amount': float(amount),
                'status': {'0': 'pending', '1': 'paid', '2': 'cancelled'}.get(status, 'pending'),
                'payment_method_name': pay_method or None,
                'payment_time': pay_time or None,
                'created_at': create_time,
            })
    return orders

def migrate():
    print("读取SQL dump...")
    with open(DUMP_FILE, 'r', encoding='utf8', errors='ignore') as f:
        sql = f.read()

    users = parse_users(sql)
    orders = parse_orders(sql)
    print(f"找到 {len(users)} 个用户, {len(orders)} 个订单")

    conn = sqlite3.connect(DB_FILE)
    cur = conn.cursor()

    # 备份提示
    print("开始迁移用户...")
    migrated = 0
    skipped = 0
    md5_count = 0

    for u in users:
        # 检查用户名是否已存在
        cur.execute("SELECT id FROM users WHERE username=?", (u['username'],))
        if cur.fetchone():
            skipped += 1
            continue

        # bcrypt $2y$ 改为 $2a$ (Go兼容格式)
        pwd = u['password']
        if pwd.startswith('$2y$'):
            pwd = '$2a$' + pwd[4:]

        is_bcrypt = u['is_bcrypt']
        if not is_bcrypt:
            md5_count += 1
            # MD5密码无法直接使用，设置一个无效hash，用户需重置
            pwd = '$2a$10$AAAAAAAAAAAAAAAAAAAAAA.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA'

        note = f"从旧系统迁移，原ID:{u['old_id']}"
        if not is_bcrypt:
            note += "，原密码为MD5格式，需要重置密码"

        cur.execute("""
            INSERT INTO users (username, email, password, is_active, is_verified, is_admin,
                balance, created_at, updated_at, last_login, notes)
            VALUES (?,?,?,?,1,0, 0.0,?,?,?,?)
        """, (
            u['username'],
            f"{u['username']}@migrate.local",
            pwd,
            1 if u['is_active'] else 0,
            u['created_at'],
            u['created_at'],
            u['last_login'],
            note,
        ))
        migrated += 1

    conn.commit()
    print(f"用户迁移完成: 成功 {migrated}, 跳过(已存在) {skipped}, MD5密码需重置 {md5_count}")

    # 迁移订单
    print("开始迁移订单...")
    order_migrated = 0
    order_skipped = 0

    for o in orders:
        # 查找对应用户
        cur.execute("SELECT id FROM users WHERE username=?", (o['user_name'],))
        row = cur.fetchone()
        if not row:
            order_skipped += 1
            continue

        user_id = row[0]

        # 检查订单是否已存在
        cur.execute("SELECT id FROM orders WHERE order_no=?", (o['order_no'],))
        if cur.fetchone():
            order_skipped += 1
            continue

        cur.execute("""
            INSERT INTO orders (order_no, user_id, package_id, amount, status,
                payment_method_name, payment_time, created_at, updated_at)
            VALUES (?,?,0,?,?,?,?,?,?)
        """, (
            o['order_no'],
            user_id,
            o['amount'],
            o['status'],
            o['payment_method_name'],
            o['payment_time'],
            o['created_at'],
            o['created_at'],
        ))
        order_migrated += 1

    conn.commit()
    conn.close()
    print(f"订单迁移完成: 成功 {order_migrated}, 跳过 {order_skipped}")
    print("\n迁移完成！")
    print(f"注意: {md5_count} 个用户使用MD5密码，需要通过'忘记密码'功能重置")

if __name__ == '__main__':
    migrate()
