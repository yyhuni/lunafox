"""
Django 环境初始化模块

在所有 Worker 脚本开头导入此模块即可自动配置 Django 环境。
"""

import os
import sys


def setup_django():
    """
    配置 Django 环境

    此函数会：
    1. 添加项目根目录到 Python 路径
    2. 设置 DJANGO_SETTINGS_MODULE 环境变量
    3. 调用 django.setup() 初始化 Django
    4. 关闭旧的数据库连接，确保使用新连接

    使用方式：
        from apps.common.django_setup import setup_django
        setup_django()
    """
    # 获取项目根目录（backend 目录）
    current_dir = os.path.dirname(os.path.abspath(__file__))
    backend_dir = os.path.join(current_dir, '../..')
    backend_dir = os.path.abspath(backend_dir)

    # 添加到 Python 路径
    if backend_dir not in sys.path:
        sys.path.insert(0, backend_dir)

    # 配置 Django
    os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'config.settings')

    # 初始化 Django
    import django
    django.setup()

    # 关闭所有旧的数据库连接，确保 Worker 进程使用新连接
    # 解决 "server closed the connection unexpectedly" 问题
    from django.db import connections
    connections.close_all()


def close_old_db_connections():
    """
    关闭旧的数据库连接

    在长时间运行的任务中调用此函数，可以确保使用有效的数据库连接。
    适用于：
    - Flow 开始前
    - Task 开始前
    - 长时间空闲后恢复操作前
    """
    from django.db import connections
    connections.close_all()


# 自动执行初始化（导入即生效）
setup_django()
