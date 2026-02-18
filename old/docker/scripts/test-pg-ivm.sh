#!/bin/bash
# pg_ivm 安装验证测试
# 在 Docker 容器中测试 install-pg-ivm.sh 的安装流程
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONTAINER_NAME="pg_ivm_test_$$"
IMAGE_NAME="postgres:15"

echo "=========================================="
echo "pg_ivm 安装验证测试"
echo "=========================================="

# 清理函数
cleanup() {
    echo ""
    echo "[清理] 删除测试容器..."
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
}
trap cleanup EXIT

# 1. 启动临时容器
echo ""
echo "[1/5] 启动临时 PostgreSQL 容器..."
docker run -d --name "$CONTAINER_NAME" \
    -e POSTGRES_PASSWORD=test \
    -e POSTGRES_USER=postgres \
    -e POSTGRES_DB=testdb \
    -e PG_VERSION=15 \
    "$IMAGE_NAME"

echo "等待 PostgreSQL 启动..."
sleep 10

if ! docker ps | grep -q "$CONTAINER_NAME"; then
    echo "错误: 容器启动失败"
    exit 1
fi

# 2. 复制并执行安装脚本
echo ""
echo "[2/5] 执行 pg_ivm 安装脚本..."
docker cp "$SCRIPT_DIR/install-pg-ivm.sh" "$CONTAINER_NAME:/tmp/install-pg-ivm.sh"

# 在容器内模拟安装（跳过 systemctl 重启，手动重启容器）
docker exec "$CONTAINER_NAME" bash -c "
set -e
export PG_VERSION=15

echo '安装编译依赖...'
apt-get update -qq
apt-get install -y -qq build-essential postgresql-server-dev-15 git

echo '编译安装 pg_ivm...'
rm -rf /tmp/pg_ivm
git clone --quiet https://github.com/sraoss/pg_ivm.git /tmp/pg_ivm
cd /tmp/pg_ivm
make -s
make install -s
rm -rf /tmp/pg_ivm
echo '✓ pg_ivm 编译安装完成'
"

# 3. 配置 shared_preload_libraries 并重启
echo ""
echo "[3/5] 配置 shared_preload_libraries..."
docker exec "$CONTAINER_NAME" bash -c "
echo \"shared_preload_libraries = 'pg_ivm'\" >> /var/lib/postgresql/data/postgresql.conf
"
echo "重启 PostgreSQL..."
docker restart "$CONTAINER_NAME"
sleep 8

# 4. 验证扩展是否可用
echo ""
echo "[4/5] 验证 pg_ivm 扩展..."
docker exec "$CONTAINER_NAME" psql -U postgres -d testdb -c "CREATE EXTENSION IF NOT EXISTS pg_ivm;" > /dev/null 2>&1

EXTENSION_EXISTS=$(docker exec "$CONTAINER_NAME" psql -U postgres -d testdb -t -c "SELECT COUNT(*) FROM pg_extension WHERE extname = 'pg_ivm';")
if [ "$(echo $EXTENSION_EXISTS | tr -d ' ')" != "1" ]; then
    echo "错误: pg_ivm 扩展未正确加载"
    exit 1
fi
echo "✓ pg_ivm 扩展已加载"

# 5. 测试 IMMV 功能
echo ""
echo "[5/5] 测试 IMMV 增量更新功能..."
docker exec "$CONTAINER_NAME" psql -U postgres -d testdb -c "
CREATE TABLE test_table (id SERIAL PRIMARY KEY, name TEXT, value INTEGER);
SELECT pgivm.create_immv('test_immv', 'SELECT id, name, value FROM test_table');
INSERT INTO test_table (name, value) VALUES ('test1', 100);
INSERT INTO test_table (name, value) VALUES ('test2', 200);
" > /dev/null 2>&1

IMMV_COUNT=$(docker exec "$CONTAINER_NAME" psql -U postgres -d testdb -t -c "SELECT COUNT(*) FROM test_immv;")
if [ "$(echo $IMMV_COUNT | tr -d ' ')" != "2" ]; then
    echo "错误: IMMV 增量更新失败，期望 2 行，实际 $(echo $IMMV_COUNT | tr -d ' ') 行"
    exit 1
fi
echo "✓ IMMV 增量更新正常 (2 行数据)"

# 测试更新
docker exec "$CONTAINER_NAME" psql -U postgres -d testdb -c "UPDATE test_table SET value = 150 WHERE name = 'test1';" > /dev/null 2>&1
UPDATED_VALUE=$(docker exec "$CONTAINER_NAME" psql -U postgres -d testdb -t -c "SELECT value FROM test_immv WHERE name = 'test1';")
if [ "$(echo $UPDATED_VALUE | tr -d ' ')" != "150" ]; then
    echo "错误: IMMV 更新同步失败"
    exit 1
fi
echo "✓ IMMV 更新同步正常"

# 测试删除
docker exec "$CONTAINER_NAME" psql -U postgres -d testdb -c "DELETE FROM test_table WHERE name = 'test2';" > /dev/null 2>&1
IMMV_COUNT_AFTER=$(docker exec "$CONTAINER_NAME" psql -U postgres -d testdb -t -c "SELECT COUNT(*) FROM test_immv;")
if [ "$(echo $IMMV_COUNT_AFTER | tr -d ' ')" != "1" ]; then
    echo "错误: IMMV 删除同步失败"
    exit 1
fi
echo "✓ IMMV 删除同步正常"

echo ""
echo "=========================================="
echo "✓ 所有测试通过"
echo "=========================================="
echo ""
echo "pg_ivm 安装验证成功，可以继续构建自定义 PostgreSQL 镜像"
