FROM postgres:15

# 安装编译依赖
RUN apt-get update && apt-get install -y \
    build-essential \
    postgresql-server-dev-15 \
    git \
    && rm -rf /var/lib/apt/lists/*

# 编译安装 pg_ivm
RUN git clone https://github.com/sraoss/pg_ivm.git /tmp/pg_ivm \
    && cd /tmp/pg_ivm \
    && make \
    && make install \
    && rm -rf /tmp/pg_ivm

# 配置 shared_preload_libraries
# 注意: 这个配置会在容器启动时被应用
RUN echo "shared_preload_libraries = 'pg_ivm'" >> /usr/share/postgresql/postgresql.conf.sample
