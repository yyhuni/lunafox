"""
截图服务

负责截图的压缩、保存和同步
"""
import io
import logging
import os
from typing import Optional

from PIL import Image

logger = logging.getLogger(__name__)


class ScreenshotService:
    """截图服务 - 负责压缩、保存和同步"""
    
    def __init__(self, max_width: int = 800, target_kb: int = 100):
        """
        初始化截图服务
        
        Args:
            max_width: 最大宽度（像素）
            target_kb: 目标文件大小（KB）
        """
        self.max_width = max_width
        self.target_kb = target_kb
    
    def compress_screenshot(self, image_path: str) -> Optional[bytes]:
        """
        压缩截图为 WebP 格式
        
        Args:
            image_path: PNG 截图文件路径
        
        Returns:
            压缩后的 WebP 二进制数据，失败返回 None
        """
        if not os.path.exists(image_path):
            logger.warning(f"截图文件不存在: {image_path}")
            return None
        
        try:
            with Image.open(image_path) as img:
                return self._compress_image(img)
        except Exception as e:
            logger.error(f"压缩截图失败: {image_path}, 错误: {e}")
            return None
    
    def compress_from_bytes(self, image_bytes: bytes) -> Optional[bytes]:
        """
        从字节数据压缩截图为 WebP 格式
        
        Args:
            image_bytes: JPEG/PNG 图片字节数据
        
        Returns:
            压缩后的 WebP 二进制数据，失败返回 None
        """
        if not image_bytes:
            return None
        
        try:
            img = Image.open(io.BytesIO(image_bytes))
            return self._compress_image(img)
        except Exception as e:
            logger.error(f"从字节压缩截图失败: {e}")
            return None
    
    def _compress_image(self, img: Image.Image) -> Optional[bytes]:
        """
        压缩 PIL Image 对象为 WebP 格式
        
        Args:
            img: PIL Image 对象
        
        Returns:
            压缩后的 WebP 二进制数据
        """
        try:
            if img.mode in ('RGBA', 'P'):
                img = img.convert('RGB')
            
            width, height = img.size
            if width > self.max_width:
                ratio = self.max_width / width
                new_size = (self.max_width, int(height * ratio))
                img = img.resize(new_size, Image.Resampling.LANCZOS)
            
            quality = 80
            while quality >= 10:
                buffer = io.BytesIO()
                img.save(buffer, format='WEBP', quality=quality, method=6)
                if len(buffer.getvalue()) <= self.target_kb * 1024:
                    return buffer.getvalue()
                quality -= 10
            
            return buffer.getvalue()
        except Exception as e:
            logger.error(f"压缩图片失败: {e}")
            return None
    
    def save_screenshot_snapshot(
        self,
        scan_id: int,
        url: str,
        image_data: bytes,
        status_code: int | None = None
    ) -> bool:
        """
        保存截图快照到 ScreenshotSnapshot 表
        
        Args:
            scan_id: 扫描 ID
            url: 截图对应的 URL
            image_data: 压缩后的图片二进制数据
            status_code: HTTP 响应状态码
        
        Returns:
            是否保存成功
        """
        from apps.asset.models import ScreenshotSnapshot
        
        try:
            ScreenshotSnapshot.objects.update_or_create(
                scan_id=scan_id,
                url=url,
                defaults={'image': image_data, 'status_code': status_code}
            )
            return True
        except Exception as e:
            logger.error(f"保存截图快照失败: scan_id={scan_id}, url={url}, 错误: {e}")
            return False
    
    def sync_screenshots_to_asset(self, scan_id: int, target_id: int) -> int:
        """
        将扫描的截图快照同步到资产表
        
        Args:
            scan_id: 扫描 ID
            target_id: 目标 ID
        
        Returns:
            同步的截图数量
        """
        from apps.asset.models import Screenshot, ScreenshotSnapshot
        
        # 使用 iterator() 避免 QuerySet 缓存大量 BinaryField 数据导致内存飙升
        # chunk_size=50: 每次只加载 50 条记录，处理完后释放内存
        snapshots = ScreenshotSnapshot.objects.filter(scan_id=scan_id).iterator(chunk_size=50)
        count = 0
        
        for snapshot in snapshots:
            try:
                Screenshot.objects.update_or_create(
                    target_id=target_id,
                    url=snapshot.url,
                    defaults={
                        'image': snapshot.image,
                        'status_code': snapshot.status_code
                    }
                )
                count += 1
            except Exception as e:
                logger.error(f"同步截图到资产表失败: url={snapshot.url}, 错误: {e}")
        
        logger.info(f"同步截图完成: scan_id={scan_id}, target_id={target_id}, 数量={count}")
        return count
    
    def process_and_save_screenshot(self, scan_id: int, url: str, image_path: str) -> bool:
        """
        处理并保存截图（压缩 + 保存快照）
        
        Args:
            scan_id: 扫描 ID
            url: 截图对应的 URL
            image_path: PNG 截图文件路径
        
        Returns:
            是否处理成功
        """
        image_data = self.compress_screenshot(image_path)
        if image_data is None:
            return False
        
        return self.save_screenshot_snapshot(scan_id, url, image_data)
