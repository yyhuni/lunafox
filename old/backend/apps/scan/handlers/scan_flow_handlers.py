"""
扫描流程处理器

负责处理扫描流程（端口扫描、子域名发现等）的状态变化和通知

职责：
- 更新各阶段的进度状态（running/completed/failed）
- 发送扫描阶段的通知
- 记录 Flow 性能指标
"""

import logging

from apps.scan.decorators import FlowContext
from apps.scan.utils.performance import FlowPerformanceTracker
from apps.scan.utils import user_log

logger = logging.getLogger(__name__)

# 存储每个 flow 的性能追踪器（使用 scan_id + stage_name 作为 key）
_flow_trackers: dict[str, FlowPerformanceTracker] = {}


def _get_tracker_key(scan_id: int, stage_name: str) -> str:
    """生成追踪器的唯一 key"""
    return f"{scan_id}_{stage_name}"


def _get_stage_from_flow_name(flow_name: str) -> str | None:
    """
    从 Flow name 获取对应的 stage

    Flow name 直接作为 stage（与 engine_config 的 key 一致）
    排除主 Flow（initiate_scan）
    """
    # 排除主 Flow，它不是阶段 Flow
    if flow_name == 'initiate_scan':
        return None
    return flow_name


def on_scan_flow_running(context: FlowContext) -> None:
    """
    扫描流程开始运行时的回调

    职责：
    - 更新阶段进度为 running
    - 发送扫描开始通知
    - 启动性能追踪

    Args:
        context: Flow 执行上下文
    """
    logger.info(
        "🚀 扫描流程开始运行 - Flow: %s, Scan ID: %s",
        context.flow_name, context.scan_id
    )

    scan_id = context.scan_id
    target_name = context.target_name or 'unknown'
    target_id = context.target_id

    # 启动性能追踪
    if scan_id:
        tracker_key = _get_tracker_key(scan_id, context.stage_name)
        tracker = FlowPerformanceTracker(context.flow_name, scan_id)
        tracker.start(target_id=target_id, target_name=target_name)
        _flow_trackers[tracker_key] = tracker

    # 更新阶段进度
    stage = _get_stage_from_flow_name(context.flow_name)
    if scan_id and stage:
        try:
            from apps.scan.services import ScanService
            service = ScanService()
            service.start_stage(scan_id, stage)
            logger.info(
                "✓ 阶段进度已更新为 running - Scan ID: %s, Stage: %s",
                scan_id, stage
            )
        except Exception as e:
            logger.error(
                "更新阶段进度失败 - Scan ID: %s, Stage: %s: %s",
                scan_id, stage, e
            )


def on_scan_flow_completed(context: FlowContext) -> None:
    """
    扫描流程完成时的回调

    职责：
    - 更新阶段进度为 completed
    - 发送扫描完成通知（可选）
    - 记录性能指标

    Args:
        context: Flow 执行上下文
    """
    logger.info(
        "✅ 扫描流程完成 - Flow: %s, Scan ID: %s",
        context.flow_name, context.scan_id
    )

    scan_id = context.scan_id
    result = context.result

    # 记录性能指标
    if scan_id:
        tracker_key = _get_tracker_key(scan_id, context.stage_name)
        tracker = _flow_trackers.pop(tracker_key, None)
        if tracker:
            tracker.finish(success=True)

    # 更新阶段进度
    stage = _get_stage_from_flow_name(context.flow_name)
    if scan_id and stage:
        try:
            from apps.scan.services import ScanService
            service = ScanService()
            # 从 flow result 中提取 detail（如果有）
            detail = None
            if isinstance(result, dict):
                detail = result.get('detail')
            service.complete_stage(scan_id, stage, detail)
            logger.info(
                "✓ 阶段进度已更新为 completed - Scan ID: %s, Stage: %s",
                scan_id, stage
            )
            # 每个阶段完成后刷新缓存统计，便于前端实时看到增量
            try:
                service.update_cached_stats(scan_id)
                logger.info("✓ 阶段完成后已刷新缓存统计 - Scan ID: %s", scan_id)
            except Exception as e:
                logger.error(
                    "阶段完成后刷新缓存统计失败 - Scan ID: %s, 错误: %s",
                    scan_id, e
                )
        except Exception as e:
            logger.error(
                "更新阶段进度失败 - Scan ID: %s, Stage: %s: %s",
                scan_id, stage, e
            )


def on_scan_flow_failed(context: FlowContext) -> None:
    """
    扫描流程失败时的回调

    职责：
    - 更新阶段进度为 failed
    - 发送扫描失败通知
    - 记录性能指标（含错误信息）
    - 写入 ScanLog 供前端显示

    Args:
        context: Flow 执行上下文
    """
    logger.info(
        "❌ 扫描流程失败 - Flow: %s, Scan ID: %s",
        context.flow_name, context.scan_id
    )

    scan_id = context.scan_id
    target_name = context.target_name or 'unknown'
    error_message = context.error_message or "未知错误"

    # 写入 ScanLog 供前端显示
    stage = _get_stage_from_flow_name(context.flow_name)
    if scan_id and stage:
        user_log(scan_id, stage, f"Failed: {error_message}", "error")

    # 记录性能指标（失败情况）
    if scan_id:
        tracker_key = _get_tracker_key(scan_id, context.stage_name)
        tracker = _flow_trackers.pop(tracker_key, None)
        if tracker:
            tracker.finish(success=False, error_message=error_message)

    # 更新阶段进度
    if scan_id and stage:
        try:
            from apps.scan.services import ScanService
            service = ScanService()
            service.fail_stage(scan_id, stage, error_message)
            logger.info(
                "✓ 阶段进度已更新为 failed - Scan ID: %s, Stage: %s",
                scan_id, stage
            )
        except Exception as e:
            logger.error(
                "更新阶段进度失败 - Scan ID: %s, Stage: %s: %s",
                scan_id, stage, e
            )

    # 发送通知
    try:
        from apps.scan.notifications import create_notification, NotificationLevel
        message = f"任务：{context.flow_name}\n状态：执行失败\n错误：{error_message}"
        create_notification(
            title=target_name,
            message=message,
            level=NotificationLevel.HIGH
        )
        logger.error(
            "✓ 扫描失败通知已发送 - Target: %s, Flow: %s, Error: %s",
            target_name, context.flow_name, error_message
        )
    except Exception as e:
        logger.error("发送扫描失败通知失败 - Flow: %s: %s", context.flow_name, e)
