# CBoard v2

Go (Gin) + Vue 3 (Naive UI) 构建的代理订阅管理面板。支持 SQLite / MySQL / PostgreSQL，开箱即用。

---

## 目录

- [安装部署](#安装部署)
- [配置说明](#配置说明)
- [管理后台使用说明](#管理后台使用说明)
- [用户端使用说明](#用户端使用说明)
- [后端 API 说明](#后端-api-说明)
- [项目结构](#项目结构)
- [常见问题](#常见问题)

---

## 安装部署

### 环境要求

- Linux：Ubuntu 20.04+、Debian 11+、CentOS 7+、AlmaLinux 8+、Rocky Linux 8+
- Go 1.22+（安装脚本自动安装）
- Node.js 18+（安装脚本自动安装）
- Nginx（反向代理，安装脚本自动配置）
- 磁盘空间 ≥ 1GB

### 方式一：无宝塔面板安装

适用于纯净 Linux 服务器。安装目录：`/opt/cboard`

```bash
bash install.sh
```
