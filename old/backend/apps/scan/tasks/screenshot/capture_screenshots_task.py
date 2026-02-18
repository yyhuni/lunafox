"""
批量截图任务

使用 Playwright 批量捕获网站截图，压缩后保存到数据库
"""
import asyncio
import logging
import time


logger = logging.getLogger(__name__)


def _run_async(coro):
    """
    在同步环境中运行异步协程
    
    Args:
        coro: 异步协程
    
    Returns:
        协程执行结果
    """
    try:
        loop = asyncio.get_event_loop()
    except RuntimeError:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
    
    return loop.run_until_complete(coro)


def _save_screenshot_with_retry(
    screenshot_service,
    scan_id: int,
    url: str,
    webp_data: bytes,
    status_code: int | None = None,
    max_retries: int = 3
) -> bool:
    """
    保存截图到数据库（带重试机制）
    
    Args:
        screenshot_service: ScreenshotService 实例
        scan_id: 扫描 ID
        url: URL
        webp_data: WebP 图片数据
        status_code: HTTP 响应状态码
        max_retries: 最大重试次数
    
    Returns:
        是否保存成功
    """
    for attempt in range(max_retries):
        try:
            if screenshot_service.save_screenshot_snapshot(scan_id, url, webp_data, status_code):
                return True
            # save 返回 False，等待后重试
            if attempt < max_retries - 1:
                wait_time = 2 ** attempt  # 指数退避：1s, 2s, 4s
                logger.warning(
                    "保存截图失败（第 %d 次尝试），%d秒后重试: %s",
                    attempt + 1, wait_time, url
                )
                time.sleep(wait_time)
        except Exception as e:
            if attempt < max_retries - 1:
                wait_time = 2 ** attempt
                logger.warning(
                    "保存截图异常（第 %d 次尝试），%d秒后重试: %s, 错误: %s",
                    attempt + 1, wait_time, url, str(e)[:100]
                )
                time.sleep(wait_time)
            else:
                logger.error("保存截图失败（已重试 %d 次）: %s", max_retries, url)
    
    return False


async def _capture_and_save_screenshots(
    urls: list[str],
    scan_id: int,
    concurrency: int
) -> dict:
    """
    异步批量截图并保存
    
    Args:
        urls: URL 列表
        scan_id: 扫描 ID
        concurrency: 并发数
    
    Returns:
        统计信息字典
    """
    from asgiref.sync import sync_to_async
    from apps.asset.services.playwright_screenshot_service import PlaywrightScreenshotService
    from apps.asset.services.screenshot_service import ScreenshotService
    
    # 初始化服务
    playwright_service = PlaywrightScreenshotService(concurrency=concurrency)
    screenshot_service = ScreenshotService()
    
    # 包装同步的保存函数为异步
    async_save_with_retry = sync_to_async(_save_screenshot_with_retry, thread_sensitive=True)
    
    # 统计
    total = len(urls)
    successful = 0
    failed = 0
    
    logger.info("开始批量截图 - URL数: %d, 并发数: %d", total, concurrency)
    
    # 批量截图
    async for url, screenshot_bytes, status_code in playwright_service.capture_batch(urls):
        if screenshot_bytes is None:
            failed += 1
            continue
        
        # 压缩为 WebP
        webp_data = screenshot_service.compress_from_bytes(screenshot_bytes)
        if webp_data is None:
            logger.warning("压缩截图失败: %s", url)
            failed += 1
            continue
        
        # 保存到数据库（带重试，使用 sync_to_async）
        if await async_save_with_retry(screenshot_service, scan_id, url, webp_data, status_code):
            successful += 1
            if successful % 10 == 0:
                logger.info("截图进度: %d/%d 成功", successful, total)
        else:
            failed += 1
    
    return {
        'total': total,
        'successful': successful,
        'failed': failed
    }



def capture_screenshots_task(
    urls: list[str],
    scan_id: int,
    target_id: int,
    config: dict
) -> dict:
    """
    批量截图任务
    
    Args:
        urls: URL 列表
        scan_id: 扫描 ID
        target_id: 目标 ID（用于日志）
        config: 截图配置
            - concurrency: 并发数（默认 5）
    
    Returns:
        dict: {
            'total': int,      # 总 URL 数
            'successful': int, # 成功截图数
            'failed': int      # 失败数
        }
    """
    if not urls:
        logger.info("URL 列表为空，跳过截图任务")
        return {'total': 0, 'successful': 0, 'failed': 0}
    
    concurrency = config.get('concurrency', 5)
    
    logger.info(
        "开始截图任务 - scan_id=%d, target_id=%d, URL数=%d, 并发=%d",
        scan_id, target_id, len(urls), concurrency
    )
    
    try:
        result = _run_async(_capture_and_save_screenshots(
            urls=urls,
            scan_id=scan_id,
            concurrency=concurrency
        ))
        
        logger.info(
            "✓ 截图任务完成 - 总数: %d, 成功: %d, 失败: %d",
            result['total'], result['successful'], result['failed']
        )
        
        return result
        
    except Exception as e:
        logger.error("截图任务失败: %s", e, exc_info=True)
        raise RuntimeError(f"截图任务失败: {e}") from e
