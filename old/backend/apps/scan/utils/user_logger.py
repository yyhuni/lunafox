"""
扫描日志记录器

提供统一的日志记录接口，用于在 Flow 中记录用户可见的扫描进度日志。

特性：
- 简单的函数式 API
- 只写入数据库（ScanLog 表），不写 Python logging
- 错误容忍（数据库失败不影响扫描执行）

职责分离：
- user_log: 用户可见日志（写数据库，前端展示）
- logger: 开发者日志（写日志文件/控制台，调试用）

使用示例：
    from apps.scan.utils import user_log
    
    # 用户日志（写数据库）
    user_log(scan_id, "port_scan", "Starting port scan")
    user_log(scan_id, "port_scan", "naabu completed: found 120 ports")
    
    # 开发者日志（写日志文件）
    logger.info("✓ 工具 %s 执行完成 - 记录数: %d", tool_name, count)
"""

import logging
from django.db import DatabaseError

logger = logging.getLogger(__name__)


def user_log(scan_id: int, stage: str, message: str, level: str = "info"):
    """
    记录用户可见的扫描日志（只写数据库）
    
    Args:
        scan_id: 扫描任务 ID
        stage: 阶段名称，如 "port_scan", "site_scan"
        message: 日志消息
        level: 日志级别，默认 "info"，可选 "warning", "error"
        
    数据库 content 格式: "[{stage}] {message}"
    """
    formatted = f"[{stage}] {message}"
    
    try:
        from apps.scan.models import ScanLog
        ScanLog.objects.create(
            scan_id=scan_id,
            level=level,
            content=formatted
        )
    except DatabaseError as e:
        logger.error("ScanLog write failed - scan_id=%s, error=%s", scan_id, e)
    except Exception as e:
        logger.error("ScanLog write unexpected error - scan_id=%s, error=%s", scan_id, e)
