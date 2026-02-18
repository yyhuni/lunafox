"""
Django ORM 实现的 Directory Repository
"""

import logging
from typing import List, Iterator
from django.db import transaction

from apps.asset.models.asset_models import Directory
from apps.asset.dtos import DirectoryDTO
from apps.common.decorators import auto_ensure_db_connection
from apps.common.utils import deduplicate_for_bulk

logger = logging.getLogger(__name__)


@auto_ensure_db_connection
class DjangoDirectoryRepository:
    """Django ORM 实现的 Directory Repository"""

    def bulk_upsert(self, items: List[DirectoryDTO]) -> int:
        """
        批量创建或更新 Directory（upsert）
        
        存在则更新所有字段，不存在则创建。
        使用 Django 原生 update_conflicts。
        
        注意：自动按模型唯一约束去重，保留最后一条记录。
        
        Args:
            items: Directory DTO 列表
            
        Returns:
            int: 处理的记录数
        """
        if not items:
            return 0
        
        try:
            # 自动按模型唯一约束去重
            unique_items = deduplicate_for_bulk(items, Directory)
            
            # 直接从 DTO 字段构建 Model
            directories = [
                Directory(
                    target_id=item.target_id,
                    url=item.url,
                    status=item.status,
                    content_length=item.content_length,
                    words=item.words,
                    lines=item.lines,
                    content_type=item.content_type or '',
                    duration=item.duration
                )
                for item in unique_items
            ]
            
            with transaction.atomic():
                Directory.objects.bulk_create(
                    directories,
                    update_conflicts=True,
                    unique_fields=['target', 'url'],
                    update_fields=[
                        'status', 'content_length', 'words',
                        'lines', 'content_type', 'duration'
                    ],
                    batch_size=1000
                )
            
            logger.debug(f"批量 upsert Directory 成功: {len(unique_items)} 条")
            return len(unique_items)
                
        except Exception as e:
            logger.error(f"批量 upsert Directory 失败: {e}")
            raise

    def bulk_create_ignore_conflicts(self, items: List[DirectoryDTO]) -> int:
        """
        批量创建 Directory（存在即跳过）
        
        与 bulk_upsert 不同，此方法不会更新已存在的记录。
        适用于批量添加场景，只提供 URL，没有其他字段数据。
        
        注意：自动按模型唯一约束去重，保留最后一条记录。
        
        Args:
            items: Directory DTO 列表
            
        Returns:
            int: 处理的记录数
        """
        if not items:
            return 0
        
        try:
            # 自动按模型唯一约束去重
            unique_items = deduplicate_for_bulk(items, Directory)
            
            directories = [
                Directory(
                    target_id=item.target_id,
                    url=item.url,
                    status=item.status,
                    content_length=item.content_length,
                    words=item.words,
                    lines=item.lines,
                    content_type=item.content_type or '',
                    duration=item.duration
                )
                for item in unique_items
            ]
            
            with transaction.atomic():
                Directory.objects.bulk_create(
                    directories,
                    ignore_conflicts=True,
                    batch_size=1000
                )
            
            logger.debug(f"批量创建 Directory 成功（ignore_conflicts）: {len(unique_items)} 条")
            return len(unique_items)
                
        except Exception as e:
            logger.error(f"批量创建 Directory 失败: {e}")
            raise

    def count_by_target(self, target_id: int) -> int:
        """统计目标下的目录总数"""
        return Directory.objects.filter(target_id=target_id).count()

    def get_all(self):
        """获取所有目录"""
        return Directory.objects.all().order_by('-created_at')

    def get_by_target(self, target_id: int):
        """获取目标下的所有目录"""
        return Directory.objects.filter(target_id=target_id).order_by('-created_at')

    def get_urls_for_export(self, target_id: int, batch_size: int = 1000) -> Iterator[str]:
        """流式导出目标下的所有目录 URL"""
        try:
            queryset = (
                Directory.objects
                .filter(target_id=target_id)
                .values_list('url', flat=True)
                .order_by('url')
                .iterator(chunk_size=batch_size)
            )
            for url in queryset:
                yield url
        except Exception as e:
            logger.error("流式导出目录 URL 失败 - Target ID: %s, 错误: %s", target_id, e)
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
            包含所有目录字段的字典
        """
        qs = (
            Directory.objects
            .filter(target_id=target_id)
            .values(
                'url', 'status', 'content_length', 'words',
                'lines', 'content_type', 'duration', 'created_at'
            )
            .order_by('url')
        )
        
        for row in qs.iterator(chunk_size=batch_size):
            yield row
