#!/bin/bash
# pg_ivm 一键安装脚本（用于远程自建 PostgreSQL 服务器）
# 要求: PostgreSQL 13+ 版本
set -e

echo "=========================================="
echo "pg_ivm 一键安装脚本"
echo "要求: PostgreSQL 13+ 版本"
echo "=========================================="
echo ""

# 检查是否以 root 运行
if [ "$EUID" -ne 0 ]; then
    echo "错误: 请使用 sudo 运行此脚本"
    exit 1
fi

# 检测 PostgreSQL 版本
detect_pg_version() {
    if command -v psql &> /dev/null; then
        psql --version | grep -oP '\d+' | head -1
    elif [ -n "$PG_VERSION" ]; then
        echo "$PG_VERSION"
    else
        echo "15"
    fi
}

PG_VERSION=${PG_VERSION:-$(detect_pg_version)}

# 检测 PostgreSQL
if ! command -v psql &> /dev/null; then
    echo "错误: 未检测到 PostgreSQL，请先安装 PostgreSQL"
    exit 1
fi

echo "检测到 PostgreSQL 版本: $PG_VERSION"

# 检查版本要求
if [ "$PG_VERSION" -lt 13 ]; then
    echo "错误: pg_ivm 要求 PostgreSQL 13+ 版本，当前版本: $PG_VERSION"
    exit 1
fi

# 安装编译依赖
echo ""
echo "[1/4] 安装编译依赖..."
if command -v apt-get &> /dev/null; then
    apt-get update -qq
    apt-get install -y -qq build-essential postgresql-server-dev-${PG_VERSION} git
elif command -v yum &> /dev/null; then
    yum install -y gcc make git postgresql${PG_VERSION}-devel
else
    echo "错误: 不支持的包管理器，请手动安装编译依赖"
    exit 1
fi
echo "✓ 编译依赖安装完成"

# 编译安装 pg_ivm
echo ""
echo "[2/4] 编译安装 pg_ivm..."
rm -rf /tmp/pg_ivm
git clone --quiet https://github.com/sraoss/pg_ivm.git /tmp/pg_ivm
cd /tmp/pg_ivm
make -s
make install -s
rm -rf /tmp/pg_ivm
echo "✓ pg_ivm 编译安装完成"

# 配置 shared_preload_libraries
echo ""
echo "[3/4] 配置 shared_preload_libraries..."

PG_CONF_DIRS=(
    "/etc/postgresql/${PG_VERSION}/main"
    "/var/lib/pgsql/${PG_VERSION}/data"
    "/var/lib/postgresql/data"
)

PG_CONF_DIR=""
for dir in "${PG_CONF_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        PG_CONF_DIR="$dir"
        break
    fi
done

if [ -z "$PG_CONF_DIR" ]; then
    echo "警告: 未找到 PostgreSQL 配置目录，请手动配置 shared_preload_libraries"
    echo "在 postgresql.conf 中添加: shared_preload_libraries = 'pg_ivm'"
else
    if grep -q "shared_preload_libraries.*pg_ivm" "$PG_CONF_DIR/postgresql.conf" 2>/dev/null; then
        echo "✓ shared_preload_libraries 已配置"
    else
        if [ -d "$PG_CONF_DIR/conf.d" ]; then
            echo "shared_preload_libraries = 'pg_ivm'" > "$PG_CONF_DIR/conf.d/pg_ivm.conf"
            echo "✓ 配置已写入 $PG_CONF_DIR/conf.d/pg_ivm.conf"
        else
            if grep -q "^shared_preload_libraries" "$PG_CONF_DIR/postgresql.conf"; then
                sed -i "s/^shared_preload_libraries = '\(.*\)'/shared_preload_libraries = '\1,pg_ivm'/" "$PG_CONF_DIR/postgresql.conf"
            else
                echo "shared_preload_libraries = 'pg_ivm'" >> "$PG_CONF_DIR/postgresql.conf"
            fi
            echo "✓ 配置已写入 $PG_CONF_DIR/postgresql.conf"
        fi
    fi
fi

# 重启 PostgreSQL
echo ""
echo "[4/4] 重启 PostgreSQL..."
if systemctl is-active --quiet postgresql; then
    systemctl restart postgresql
    echo "✓ PostgreSQL 已重启"
elif systemctl is-active --quiet postgresql-${PG_VERSION}; then
    systemctl restart postgresql-${PG_VERSION}
    echo "✓ PostgreSQL 已重启"
else
    echo "警告: 无法自动重启 PostgreSQL，请手动重启"
fi

echo ""
echo "=========================================="
echo "✓ pg_ivm 安装完成"
echo "=========================================="
echo ""
echo "验证安装:"
echo "  psql -U postgres -c \"CREATE EXTENSION IF NOT EXISTS pg_ivm;\""
echo ""
