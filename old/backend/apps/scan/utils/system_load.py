"""
系统负载检查工具

提供统一的系统负载检查功能，用于：
- Flow 入口处检查系统资源是否充足
- 防止在高负载时启动新的扫描任务
"""

import logging
import time

import psutil
from django.conf import settings

logger = logging.getLogger(__name__)

# 动态并发控制阈值（可在 Django settings 中覆盖）
SCAN_CPU_HIGH: float = getattr(settings, 'SCAN_CPU_HIGH', 90.0)
SCAN_MEM_HIGH: float = getattr(settings, 'SCAN_MEM_HIGH', 80.0)
SCAN_LOAD_CHECK_INTERVAL: int = getattr(settings, 'SCAN_LOAD_CHECK_INTERVAL', 180)


def _get_current_load() -> tuple[float, float]:
    """获取当前 CPU 和内存使用率"""
    return psutil.cpu_percent(interval=0.5), psutil.virtual_memory().percent


def wait_for_system_load(
    cpu_threshold: float = SCAN_CPU_HIGH,
    mem_threshold: float = SCAN_MEM_HIGH,
    check_interval: int = SCAN_LOAD_CHECK_INTERVAL,
    context: str = "task"
) -> None:
    """
    等待系统负载降到阈值以下

    在高负载时阻塞等待，直到 CPU 和内存都低于阈值。
    用于 Flow 入口处，防止在资源紧张时启动新任务。
    """
    while True:
        cpu, mem = _get_current_load()

        if cpu < cpu_threshold and mem < mem_threshold:
            logger.debug(
                "[%s] 系统负载正常: cpu=%.1f%%, mem=%.1f%%",
                context, cpu, mem
            )
            return

        logger.info(
            "[%s] 系统负载较高，等待资源释放: "
            "cpu=%.1f%% (阈值 %.1f%%), mem=%.1f%% (阈值 %.1f%%)",
            context, cpu, cpu_threshold, mem, mem_threshold
        )
        time.sleep(check_interval)


def check_system_load(
    cpu_threshold: float = SCAN_CPU_HIGH,
    mem_threshold: float = SCAN_MEM_HIGH
) -> dict:
    """
    检查当前系统负载（非阻塞）

    Returns:
        dict: cpu_percent, mem_percent, cpu_threshold, mem_threshold, is_overloaded
    """
    cpu, mem = _get_current_load()

    return {
        'cpu_percent': cpu,
        'mem_percent': mem,
        'cpu_threshold': cpu_threshold,
        'mem_threshold': mem_threshold,
        'is_overloaded': cpu >= cpu_threshold or mem >= mem_threshold,
    }

