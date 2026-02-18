"""
扫描流程装饰器模块

提供轻量级的 @scan_flow 和 @scan_task 装饰器，替代 Prefect 的 @flow 和 @task。

核心功能：
- @scan_flow: 状态管理、通知、性能追踪
- @scan_task: 重试逻辑（大部分 task 不需要重试，可直接移除装饰器）

设计原则：
- 保持与 Prefect 装饰器相同的使用方式
- 零依赖，无额外内存开销
- 保留原函数签名和返回值
"""

import functools
import logging
import time
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any, Callable, Optional

logger = logging.getLogger(__name__)


@dataclass
class FlowContext:
    """
    Flow 执行上下文

    替代 Prefect 的 Flow、FlowRun、State 参数，传递给回调函数。
    """
    flow_name: str
    stage_name: str
    scan_id: Optional[int] = None
    target_id: Optional[int] = None
    target_name: Optional[str] = None
    parameters: dict = field(default_factory=dict)
    start_time: datetime = field(default_factory=datetime.now)
    end_time: Optional[datetime] = None
    result: Any = None
    error: Optional[Exception] = None
    error_message: Optional[str] = None


def scan_flow(
    name: Optional[str] = None,
    stage_name: Optional[str] = None,
    on_running: Optional[list[Callable]] = None,
    on_completion: Optional[list[Callable]] = None,
    on_failure: Optional[list[Callable]] = None,
    log_prints: bool = True,  # 保持与 Prefect 兼容，但不使用
):
    """
    扫描流程装饰器

    替代 Prefect 的 @flow 装饰器，提供：
    - 自动状态管理（start_stage/complete_stage/fail_stage）
    - 生命周期回调（on_running/on_completion/on_failure）
    - 性能追踪（FlowPerformanceTracker）
    - 失败通知

    Args:
        name: Flow 名称，默认使用函数名
        stage_name: 阶段名称，默认使用 name
        on_running: 流程开始时的回调列表
        on_completion: 流程完成时的回调列表
        on_failure: 流程失败时的回调列表
        log_prints: 保持与 Prefect 兼容，不使用

    Usage:
        @scan_flow(name="site_scan", on_running=[on_scan_flow_running])
        def site_scan_flow(scan_id: int, target_id: int, ...):
            ...
    """
    def decorator(func: Callable) -> Callable:
        flow_name = name or func.__name__
        actual_stage_name = stage_name or flow_name

        @functools.wraps(func)
        def wrapper(*args, **kwargs) -> Any:
            # 提取参数
            scan_id = kwargs.get('scan_id')
            target_id = kwargs.get('target_id')
            target_name = kwargs.get('target_name')

            # 创建上下文
            context = FlowContext(
                flow_name=flow_name,
                stage_name=actual_stage_name,
                scan_id=scan_id,
                target_id=target_id,
                target_name=target_name,
                parameters=kwargs.copy(),
                start_time=datetime.now(),
            )

            # 执行 on_running 回调
            if on_running:
                for callback in on_running:
                    try:
                        callback(context)
                    except Exception as e:
                        logger.warning("on_running 回调执行失败: %s", e)

            try:
                # 执行原函数
                result = func(*args, **kwargs)

                # 更新上下文
                context.end_time = datetime.now()
                context.result = result

                # 执行 on_completion 回调
                if on_completion:
                    for callback in on_completion:
                        try:
                            callback(context)
                        except Exception as e:
                            logger.warning("on_completion 回调执行失败: %s", e)

                return result

            except Exception as e:
                # 更新上下文
                context.end_time = datetime.now()
                context.error = e
                context.error_message = str(e)

                # 执行 on_failure 回调
                if on_failure:
                    for callback in on_failure:
                        try:
                            callback(context)
                        except Exception as cb_error:
                            logger.warning("on_failure 回调执行失败: %s", cb_error)

                # 重新抛出异常
                raise

        return wrapper
    return decorator


def scan_task(
    retries: int = 0,
    retry_delay: float = 1.0,
    name: Optional[str] = None,  # 保持与 Prefect 兼容
):
    """
    扫描任务装饰器

    替代 Prefect 的 @task 装饰器，提供重试能力。

    注意：当前代码中大部分 @task 都是 retries=0，可以直接移除装饰器。
    只有需要重试的 task 才需要使用此装饰器。

    Args:
        retries: 失败后重试次数，默认 0（不重试）
        retry_delay: 重试间隔（秒），默认 1.0
        name: 任务名称，保持与 Prefect 兼容，不使用

    Usage:
        @scan_task(retries=3, retry_delay=2.0)
        def run_scan_tool(command: str, timeout: int):
            ...
    """
    def decorator(func: Callable) -> Callable:
        task_name = name or func.__name__

        @functools.wraps(func)
        def wrapper(*args, **kwargs) -> Any:
            last_exception = None

            for attempt in range(retries + 1):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    last_exception = e
                    if attempt < retries:
                        logger.warning(
                            "任务 %s 重试 %d/%d: %s",
                            task_name, attempt + 1, retries, e
                        )
                        time.sleep(retry_delay)
                    else:
                        logger.error(
                            "任务 %s 重试耗尽 (%d 次): %s",
                            task_name, retries + 1, e
                        )

            # 重试耗尽，抛出最后一个异常
            raise last_exception

        # 添加 submit 方法以保持与 Prefect task.submit() 的兼容性
        # 注意：这只是为了迁移过渡，最终应该使用 ThreadPoolExecutor
        wrapper.fn = func

        return wrapper
    return decorator
