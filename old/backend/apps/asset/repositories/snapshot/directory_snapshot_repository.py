"""Directory Snapshot Repository - 目录快照数据访问层"""

import logging
from typing import List, Iterator
from django.db import transaction

from apps.asset.models import DirectorySnapshot
from apps.asset.dtos.snapshot import DirectorySnapshotDTO
from apps.common.decorators import auto_ensure_db_connection
from apps.common.utils import deduplicate_for_bulk

logger = logging.getLogger(__name__)


@auto_ensure_db_connection
class DjangoDirectorySnapshotRepository:
    """
    目录快照仓储（Django ORM 实现）
    
    负责目录快照表的数据访问操作
    """
    
    def save_snapshots(self, items: List[DirectorySnapshotDTO]) -> None:
        """
        批量保存目录快照记录
        
        使用 ignore_conflicts 策略，如果快照已存在（相同 scan + url）则跳过
        
        注意：会自动按 (scan_id, url) 去重，保留最后一条记录。
        
        Args:
            items: 目录快照 DTO 列表
        
        Raises:
            ValueError: items 为空
            Exception: 数据库操作失败
        """
        if not items:
            logger.warning("目录快照列表为空，跳过保存")
            return
        
        try:
            # 根据模型唯一约束自动去重
            unique_items = deduplicate_for_bulk(items, DirectorySnapshot)
            
            # 转换为 Django 模型对象
            snapshot_objects = [
                DirectorySnapshot(
                    scan_id=item.scan_id,
                    url=item.url,
                    status=item.status,
                    content_length=item.content_length,
                    words=item.words,
                    lines=item.lines,
                    content_type=item.content_type,
                    duration=item.duration
                )
                for item in unique_items
            ]
            
            with transaction.atomic():
                # 批量插入，忽略冲突
                # 如果 scan + url 已存在，跳过
                DirectorySnapshot.objects.bulk_create(
                    snapshot_objects,
                    ignore_conflicts=True
                )
            
            logger.debug("成功保存 %d 条目录快照记录", len(unique_items))
            
        except Exception as e:
            logger.error(
                "批量保存目录快照失败 - 数量: %d, 错误: %s",
                len(items),
                str(e),
                exc_info=True
            )
            raise
    
    def get_by_scan(self, scan_id: int):
        return DirectorySnapshot.objects.filter(scan_id=scan_id).order_by('-created_at')

    def get_all(self):
        return DirectorySnapshot.objects.all().order_by('-created_at')

    def iter_raw_data_for_export(
        self, 
        scan_id: int,
        batch_size: int = 1000
    ) -> Iterator[dict]:
        """
        流式获取原始数据用于 CSV 导出
        
        Args:
            scan_id: 扫描 ID
            batch_size: 每批数据量
        
        Yields:
            包含所有目录字段的字典
        """
        qs = (
            DirectorySnapshot.objects
            .filter(scan_id=scan_id)
            .values(
                'url', 'status', 'content_length', 'words',
                'lines', 'content_type', 'duration', 'created_at'
            )
            .order_by('url')
        )
        
        for row in qs.iterator(chunk_size=batch_size):
            yield row
