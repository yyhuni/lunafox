"""
截图 Flow

负责编排截图的完整流程：
1. 从 Provider 获取 URL 列表
2. 批量截图并保存快照
3. 同步到资产表
"""

import logging

from apps.scan.decorators import scan_flow
from apps.scan.handlers.scan_flow_handlers import (
    on_scan_flow_completed,
    on_scan_flow_failed,
    on_scan_flow_running,
)
from apps.scan.providers import TargetProvider
from apps.scan.tasks.screenshot import capture_screenshots_task
from apps.scan.utils import user_log, wait_for_system_load

logger = logging.getLogger(__name__)


def _parse_screenshot_config(enabled_tools: dict) -> dict:
    """解析截图配置"""
    playwright_config = enabled_tools.get('playwright', {})
    return {
        'concurrency': playwright_config.get('concurrency', 5),
    }


def _collect_urls_from_provider(provider: TargetProvider) -> tuple[list[str], str]:
    """
    从 Provider 收集网站 URL（带回退逻辑）

    优先级：WebSite → HostPortMapping → Default URL

    Returns:
        tuple: (urls, source)
            - urls: URL 列表
            - source: 数据来源 ('website' | 'host_port' | 'default')
    """
    logger.info("从 Provider 获取网站 URL - Provider: %s", type(provider).__name__)

    # 优先从 WebSite 获取
    urls = list(provider.iter_websites())
    if urls:
        logger.info("使用 WebSite 数据源 - 数量: %d", len(urls))
        return urls, "website"

    # 回退到 HostPortMapping
    urls = list(provider.iter_host_port_urls())
    if urls:
        logger.info("WebSite 为空，回退到 HostPortMapping - 数量: %d", len(urls))
        return urls, "host_port"

    # 最终回退到默认 URL
    urls = list(provider.iter_default_urls())
    logger.info("HostPortMapping 为空，回退到默认 URL - 数量: %d", len(urls))
    return urls, "default"


def _build_empty_result(scan_id: int, target_name: str) -> dict:
    """构建空结果"""
    return {
        'success': True,
        'scan_id': scan_id,
        'target': target_name,
        'total_urls': 0,
        'successful': 0,
        'failed': 0,
        'synced': 0
    }


@scan_flow(
    name="screenshot",
    on_running=[on_scan_flow_running],
    on_completion=[on_scan_flow_completed],
    on_failure=[on_scan_flow_failed],
)
def screenshot_flow(
    scan_id: int,
    target_id: int,
    scan_workspace_dir: str,
    enabled_tools: dict,
    provider: TargetProvider,
) -> dict:
    """
    截图 Flow

    Args:
        scan_id: 扫描任务 ID
        target_id: 目标 ID
        scan_workspace_dir: 扫描工作空间目录
        enabled_tools: 启用的工具配置
        provider: TargetProvider 实例

    Returns:
        截图结果字典
    """
    try:
        wait_for_system_load(context="screenshot_flow")

        # 从 provider 获取 target_name
        target_name = provider.get_target_name()
        if not target_name:
            raise ValueError("无法获取 Target 名称")

        logger.info(
            "开始截图扫描 - Scan ID: %s, Target: %s",
            scan_id, target_name
        )
        user_log(scan_id, "screenshot", "Starting screenshot capture")

        # Step 1: 解析配置
        config = _parse_screenshot_config(enabled_tools)
        concurrency = config['concurrency']
        logger.info("截图配置 - 并发: %d", concurrency)

        # Step 2: 从 Provider 收集 URL 列表（带回退逻辑）
        urls, source = _collect_urls_from_provider(provider)
        logger.info("URL 收集完成 - 来源: %s, 数量: %d", source, len(urls))

        if not urls:
            logger.warning("没有可截图的 URL，跳过截图任务")
            user_log(scan_id, "screenshot", "Skipped: no URLs to capture", "warning")
            return _build_empty_result(scan_id, target_name)

        user_log(scan_id, "screenshot", f"Found {len(urls)} URLs to capture")

        # Step 3: 批量截图
        logger.info("批量截图 - %d 个 URL", len(urls))
        capture_result = capture_screenshots_task(
            urls=urls,
            scan_id=scan_id,
            target_id=target_id,
            config={'concurrency': concurrency}
        )

        # Step 4: 同步到资产表
        logger.info("同步截图到资产表")
        from apps.asset.services.screenshot_service import ScreenshotService
        synced = ScreenshotService().sync_screenshots_to_asset(scan_id, target_id)

        total = capture_result['total']
        successful = capture_result['successful']
        failed = capture_result['failed']

        logger.info(
            "✓ 截图完成 - 总数: %d, 成功: %d, 失败: %d, 同步: %d",
            total, successful, failed, synced
        )
        user_log(
            scan_id, "screenshot",
            f"Screenshot completed: {successful}/{total} captured, {synced} synced"
        )

        return {
            'success': True,
            'scan_id': scan_id,
            'target': target_name,
            'total_urls': total,
            'successful': successful,
            'failed': failed,
            'synced': synced
        }

    except Exception:
        logger.exception("截图 Flow 失败")
        user_log(scan_id, "screenshot", "Screenshot failed", "error")
        raise
