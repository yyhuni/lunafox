"""
Django ORM 实现的 WebSite Repository
"""

import logging
from typing import List, Generator, Optional, Iterator
from django.db import transaction

from apps.asset.models.asset_models import WebSite
from apps.asset.dtos import WebSiteDTO
from apps.common.decorators import auto_ensure_db_connection
from apps.common.utils import deduplicate_for_bulk

logger = logging.getLogger(__name__)


@auto_ensure_db_connection
class DjangoWebSiteRepository:
    """Django ORM 实现的 WebSite Repository"""

    def bulk_upsert(self, items: List[WebSiteDTO]) -> int:
        """
        批量创建或更新 WebSite（upsert）
        
        存在则更新所有字段，不存在则创建。
        使用 Django 原生 update_conflicts。
        
        注意：自动按模型唯一约束去重，保留最后一条记录。
        
        Args:
            items: WebSite DTO 列表
            
        Returns:
            int: 处理的记录数
        """
        if not items:
            return 0
        
        try:
            # 自动按模型唯一约束去重
            unique_items = deduplicate_for_bulk(items, WebSite)
            
            # 直接从 DTO 字段构建 Model
            websites = [
                WebSite(
                    target_id=item.target_id,
                    url=item.url,
                    host=item.host or '',
                    location=item.location or '',
                    title=item.title or '',
                    webserver=item.webserver or '',
                    response_body=item.response_body or '',
                    content_type=item.content_type or '',
                    tech=item.tech if item.tech else [],
                    status_code=item.status_code,
                    content_length=item.content_length,
                    vhost=item.vhost,
                    response_headers=item.response_headers if item.response_headers else ''
                )
                for item in unique_items
            ]
            
            with transaction.atomic():
                WebSite.objects.bulk_create(
                    websites,
                    update_conflicts=True,
                    unique_fields=['url', 'target'],
                    update_fields=[
                        'host', 'location', 'title', 'webserver',
                        'response_body', 'content_type', 'tech',
                        'status_code', 'content_length', 'vhost', 'response_headers'
                    ],
                    batch_size=1000
                )
            
            logger.debug(f"批量 upsert WebSite 成功: {len(unique_items)} 条")
            return len(unique_items)
                
        except Exception as e:
            logger.error(f"批量 upsert WebSite 失败: {e}")
            raise

    def get_urls_for_export(self, target_id: int, batch_size: int = 1000) -> Generator[str, None, None]:
        """
        流式导出目标下的所有站点 URL
        """
        try:
            queryset = WebSite.objects.filter(
                target_id=target_id
            ).values_list('url', flat=True).iterator(chunk_size=batch_size)
            
            for url in queryset:
                yield url
        except Exception as e:
            logger.error(f"流式导出站点 URL 失败 - Target ID: {target_id}, 错误: {e}")
            raise

    def get_all(self):
        """获取所有网站"""
        return WebSite.objects.all().order_by('-created_at')

    def get_by_target(self, target_id: int):
        """获取目标下的所有网站"""
        return WebSite.objects.filter(target_id=target_id).order_by('-created_at')

    def count_by_target(self, target_id: int) -> int:
        """统计目标下的站点总数"""
        return WebSite.objects.filter(target_id=target_id).count()

    def get_by_url(self, url: str, target_id: int) -> Optional[int]:
        """根据 URL 和 target_id 查找站点 ID"""
        website = WebSite.objects.filter(url=url, target_id=target_id).first()
        return website.id if website else None

    def bulk_create_ignore_conflicts(self, items: List[WebSiteDTO]) -> int:
        """
        批量创建 WebSite（存在即跳过）
        
        注意：自动按模型唯一约束去重，保留最后一条记录。
        """
        if not items:
            return 0
        
        try:
            # 自动按模型唯一约束去重
            unique_items = deduplicate_for_bulk(items, WebSite)
            
            websites = [
                WebSite(
                    target_id=item.target_id,
                    url=item.url,
                    host=item.host or '',
                    location=item.location or '',
                    title=item.title or '',
                    webserver=item.webserver or '',
                    response_body=item.response_body or '',
                    content_type=item.content_type or '',
                    tech=item.tech if item.tech else [],
                    status_code=item.status_code,
                    content_length=item.content_length,
                    vhost=item.vhost,
                    response_headers=item.response_headers if item.response_headers else ''
                )
                for item in unique_items
            ]
            
            with transaction.atomic():
                WebSite.objects.bulk_create(
                    websites,
                    ignore_conflicts=True,
                    batch_size=1000
                )
            
            logger.debug(f"批量创建 WebSite 成功（ignore_conflicts）: {len(unique_items)} 条")
            return len(unique_items)
                
        except Exception as e:
            logger.error(f"批量创建 WebSite 失败: {e}")
            raise

    def iter_raw_data_for_export(
        self, 
        target_id: int,
        batch_size: int = 1000
    ) -> Iterator[dict]:
        """
        流式获取原始数据用于 CSV 导出
        
        Args:
            target_id: 目标 ID
            batch_size: 每批数据量
        
        Yields:
            包含所有网站字段的字典
        """
        qs = (
            WebSite.objects
            .filter(target_id=target_id)
            .values(
                'url', 'host', 'location', 'title', 'status_code',
                'content_length', 'content_type', 'webserver', 'tech',
                'response_body', 'response_headers', 'vhost', 'created_at'
            )
            .order_by('url')
        )
        
        for row in qs.iterator(chunk_size=batch_size):
            yield row
