#!/bin/bash
set -e

# 创建应用数据库（生产 + 开发）
# 使用条件创建避免与 POSTGRES_DB 自动创建的数据库冲突
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" <<-EOSQL
	SELECT 'CREATE DATABASE xingrin' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'xingrin')\gexec
	SELECT 'CREATE DATABASE xingrin_dev' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'xingrin_dev')\gexec
	GRANT ALL PRIVILEGES ON DATABASE xingrin TO "$POSTGRES_USER";
	GRANT ALL PRIVILEGES ON DATABASE xingrin_dev TO "$POSTGRES_USER";
EOSQL

# 启用 pg_trgm 扩展（用于文本模糊搜索索引）
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "xingrin" <<-EOSQL
	CREATE EXTENSION IF NOT EXISTS pg_trgm;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "xingrin_dev" <<-EOSQL
	CREATE EXTENSION IF NOT EXISTS pg_trgm;
EOSQL
