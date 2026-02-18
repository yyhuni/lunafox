"""
Playwright 截图服务

使用 Playwright 异步批量捕获网站截图
"""
import asyncio
import logging
from typing import Optional, AsyncGenerator

logger = logging.getLogger(__name__)


class PlaywrightScreenshotService:
    """Playwright 截图服务 - 异步多 Page 并发截图"""
    
    # 内置默认值（不暴露给用户）
    DEFAULT_VIEWPORT_WIDTH = 1920
    DEFAULT_VIEWPORT_HEIGHT = 1080
    DEFAULT_TIMEOUT = 30000  # 毫秒
    DEFAULT_JPEG_QUALITY = 85
    
    def __init__(
        self,
        viewport_width: int = DEFAULT_VIEWPORT_WIDTH,
        viewport_height: int = DEFAULT_VIEWPORT_HEIGHT,
        timeout: int = DEFAULT_TIMEOUT,
        concurrency: int = 5
    ):
        """
        初始化 Playwright 截图服务
        
        Args:
            viewport_width: 视口宽度（像素）
            viewport_height: 视口高度（像素）
            timeout: 页面加载超时时间（毫秒）
            concurrency: 并发截图数
        """
        self.viewport_width = viewport_width
        self.viewport_height = viewport_height
        self.timeout = timeout
        self.concurrency = concurrency
    
    async def capture_screenshot(self, url: str, page) -> tuple[Optional[bytes], Optional[int]]:
        """
        捕获单个 URL 的截图
        
        Args:
            url: 目标 URL
            page: Playwright Page 对象
        
        Returns:
            (screenshot_bytes, status_code) 元组
            - screenshot_bytes: JPEG 格式的截图字节数据，失败返回 None
            - status_code: HTTP 响应状态码，失败返回 None
        """
        status_code = None
        try:
            # 尝试加载页面，即使返回错误状态码也继续截图
            try:
                response = await page.goto(url, timeout=self.timeout, wait_until='networkidle')
                if response:
                    status_code = response.status
            except Exception as goto_error:
                # 页面加载失败（4xx/5xx 或其他错误），但页面可能已部分渲染
                # 仍然尝试截图以捕获错误页面
                logger.debug("页面加载异常但尝试截图: %s, 错误: %s", url, str(goto_error)[:50])
            
            # 尝试截图（即使 goto 失败）
            screenshot_bytes = await page.screenshot(
                type='jpeg',
                quality=self.DEFAULT_JPEG_QUALITY,
                full_page=False
            )
            return (screenshot_bytes, status_code)
        except asyncio.TimeoutError:
            logger.warning("截图超时: %s", url)
            return (None, None)
        except Exception as e:
            logger.warning("截图失败: %s, 错误: %s", url, str(e)[:100])
            return (None, None)
    
    async def _capture_with_semaphore(
        self,
        url: str,
        context,
        semaphore: asyncio.Semaphore
    ) -> tuple[str, Optional[bytes], Optional[int]]:
        """
        使用信号量控制并发的截图任务
        
        Args:
            url: 目标 URL
            context: Playwright BrowserContext
            semaphore: 并发控制信号量
        
        Returns:
            (url, screenshot_bytes, status_code) 元组
        """
        async with semaphore:
            page = await context.new_page()
            try:
                screenshot_bytes, status_code = await self.capture_screenshot(url, page)
                return (url, screenshot_bytes, status_code)
            finally:
                await page.close()
    
    async def capture_batch(
        self,
        urls: list[str]
    ) -> AsyncGenerator[tuple[str, Optional[bytes], Optional[int]], None]:
        """
        批量捕获截图（异步生成器）
        
        使用单个 BrowserContext + 多 Page 并发模式
        通过 Semaphore 控制并发数
        
        Args:
            urls: URL 列表
        
        Yields:
            (url, screenshot_bytes, status_code) 元组
        """
        if not urls:
            return
        
        from playwright.async_api import async_playwright
        
        async with async_playwright() as p:
            # 启动浏览器（headless 模式）
            browser = await p.chromium.launch(
                headless=True,
                args=[
                    '--no-sandbox',
                    '--disable-setuid-sandbox',
                    '--disable-dev-shm-usage',
                    '--disable-gpu'
                ]
            )
            
            try:
                # 创建单个 context
                context = await browser.new_context(
                    viewport={
                        'width': self.viewport_width,
                        'height': self.viewport_height
                    },
                    ignore_https_errors=True,
                    user_agent='Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
                )
                
                # 使用 Semaphore 控制并发
                semaphore = asyncio.Semaphore(self.concurrency)
                
                # 创建所有任务
                tasks = [
                    self._capture_with_semaphore(url, context, semaphore)
                    for url in urls
                ]
                
                # 使用 as_completed 实现流式返回
                for coro in asyncio.as_completed(tasks):
                    result = await coro
                    yield result
                
                await context.close()
                
            finally:
                await browser.close()
    
    async def capture_batch_collect(
        self,
        urls: list[str]
    ) -> list[tuple[str, Optional[bytes], Optional[int]]]:
        """
        批量捕获截图（收集所有结果）
        
        Args:
            urls: URL 列表
        
        Returns:
            [(url, screenshot_bytes, status_code), ...] 列表
        """
        results = []
        async for result in self.capture_batch(urls):
            results.append(result)
        return results
