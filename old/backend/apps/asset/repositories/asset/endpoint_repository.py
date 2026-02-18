"""Endpoint Repository - Django ORM 实现"""

import logging
from typing import List, Iterator

from apps.asset.models import Endpoint
from apps.asset.dtos.asset import EndpointDTO
from apps.common.decorators import auto_ensure_db_connection
from apps.common.utils import deduplicate_for_bulk
from django.db import transaction

logger = logging.getLogger(__name__)


@auto_ensure_db_connection
class DjangoEndpointRepository:
    """端点 Repository - 负责端点表的数据访问"""
    
    def bulk_upsert(self, items: List[EndpointDTO]) -> int:
        """
        批量创建或更新端点（upsert）
        
        存在则更新所有字段，不存在则创建。
        使用 Django 原生 update_conflicts。
        
        注意：自动按模型唯一约束去重，保留最后一条记录。
        
        Args:
            items: 端点 DTO 列表
            
        Returns:
            int: 处理的记录数
        """
        if not items:
            return 0
        
        try:
            # 自动按模型唯一约束去重
            unique_items = deduplicate_for_bulk(items, Endpoint)
            
            # 直接从 DTO 字段构建 Model
            endpoints = [
                Endpoint(
                    target_id=item.target_id,
                    url=item.url,
                    host=item.host or '',
                    title=item.title or '',
                    status_code=item.status_code,
                    content_length=item.content_length,
                    webserver=item.webserver or '',
                    response_body=item.response_body or '',
                    content_type=item.content_type or '',
                    tech=item.tech if item.tech else [],
                    vhost=item.vhost,
                    location=item.location or '',
                    matched_gf_patterns=item.matched_gf_patterns if item.matched_gf_patterns else [],
                    response_headers=item.response_headers if item.response_headers else ''
                )
                for item in unique_items
            ]
            
            with transaction.atomic():
                Endpoint.objects.bulk_create(
                    endpoints,
                    update_conflicts=True,
                    unique_fields=['url', 'target'],
                    update_fields=[
                        'host', 'title', 'status_code', 'content_length',
                        'webserver', 'response_body', 'content_type', 'tech',
                        'vhost', 'location', 'matched_gf_patterns', 'response_headers'
                    ],
                    batch_size=1000
                )
            
            logger.debug(f"批量 upsert 端点成功: {len(unique_items)} 条")
            return len(unique_items)
                
        except Exception as e:
            logger.error(f"批量 upsert 端点失败: {e}")
            raise
    
    def get_all(self):
        """获取所有端点（全局查询）"""
        return Endpoint.objects.all().order_by('-created_at')
    
    def get_by_target(self, target_id: int):
        """
        获取目标下的所有端点
        
        Args:
            target_id: 目标 ID
            
        Returns:
            QuerySet: 端点查询集
        """
        return Endpoint.objects.filter(target_id=target_id).order_by('-created_at')
    
    def count_by_target(self, target_id: int) -> int:
        """
        统计目标下的端点数量
        
        Args:
            target_id: 目标 ID
            
        Returns:
            int: 端点数量
        """
        return Endpoint.objects.filter(target_id=target_id).count()

    def bulk_create_ignore_conflicts(self, items: List[EndpointDTO]) -> int:
        """
        批量创建端点（存在即跳过）
        
        与 bulk_upsert 不同，此方法不会更新已存在的记录。
        适用于快速扫描场景，只提供 URL，没有其他字段数据。
        
        注意：自动按模型唯一约束去重，保留最后一条记录。
        
        Args:
            items: 端点 DTO 列表
            
        Returns:
            int: 处理的记录数
        """
        if not items:
            return 0
        
        try:
            # 自动按模型唯一约束去重
            unique_items = deduplicate_for_bulk(items, Endpoint)
            
            # 直接从 DTO 字段构建 Model
            endpoints = [
                Endpoint(
                    target_id=item.target_id,
                    url=item.url,
                    host=item.host or '',
                    title=item.title or '',
                    status_code=item.status_code,
                    content_length=item.content_length,
                    webserver=item.webserver or '',
                    response_body=item.response_body or '',
                    content_type=item.content_type or '',
                    tech=item.tech if item.tech else [],
                    vhost=item.vhost,
                    location=item.location or '',
                    matched_gf_patterns=item.matched_gf_patterns if item.matched_gf_patterns else [],
                    response_headers=item.response_headers if item.response_headers else ''
                )
                for item in unique_items
            ]
            
            with transaction.atomic():
                Endpoint.objects.bulk_create(
                    endpoints,
                    ignore_conflicts=True,
                    batch_size=1000
                )
            
            logger.debug(f"批量创建端点成功（ignore_conflicts）: {len(unique_items)} 条")
            return len(unique_items)
                
        except Exception as e:
            logger.error(f"批量创建端点失败: {e}")
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
            包含所有端点字段的字典
        """
        qs = (
            Endpoint.objects
            .filter(target_id=target_id)
            .values(
                'url', 'host', 'location', 'title', 'status_code',
                'content_length', 'content_type', 'webserver', 'tech',
                'response_body', 'response_headers', 'vhost', 'matched_gf_patterns', 'created_at'
            )
            .order_by('url')
        )
        
        for row in qs.iterator(chunk_size=batch_size):
            yield row
