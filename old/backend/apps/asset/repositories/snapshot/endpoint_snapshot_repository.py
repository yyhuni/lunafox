"""EndpointSnapshot Repository - Django ORM 实现"""

import logging
from typing import List, Iterator

from apps.asset.models.snapshot_models import EndpointSnapshot
from apps.asset.dtos.snapshot import EndpointSnapshotDTO
from apps.common.decorators import auto_ensure_db_connection
from apps.common.utils import deduplicate_for_bulk

logger = logging.getLogger(__name__)


@auto_ensure_db_connection
class DjangoEndpointSnapshotRepository:
    """端点快照 Repository - 负责端点快照表的数据访问"""

    def save_snapshots(self, items: List[EndpointSnapshotDTO]) -> None:
        """
        保存端点快照
        
        注意：会自动按 (scan_id, url) 去重，保留最后一条记录。
        
        Args:
            items: 端点快照 DTO 列表
        
        Note:
            - 保存完整的快照数据
            - 基于唯一约束 (scan + url) 自动去重
        """
        try:
            logger.debug("准备保存端点快照 - 数量: %d", len(items))
            
            if not items:
                logger.debug("端点快照为空，跳过保存")
                return
            
            # 根据模型唯一约束自动去重
            unique_items = deduplicate_for_bulk(items, EndpointSnapshot)
                
            # 构建快照对象
            snapshots = []
            for item in unique_items:
                snapshots.append(EndpointSnapshot(
                    scan_id=item.scan_id,
                    url=item.url,
                    host=item.host if item.host else '',
                    title=item.title,
                    status_code=item.status_code,
                    content_length=item.content_length,
                    location=item.location,
                    webserver=item.webserver,
                    content_type=item.content_type,
                    tech=item.tech if item.tech else [],
                    response_body=item.response_body,
                    vhost=item.vhost,
                    matched_gf_patterns=item.matched_gf_patterns if item.matched_gf_patterns else [],
                    response_headers=item.response_headers if item.response_headers else ''
                ))
            
            # 批量创建（忽略冲突，基于唯一约束去重）
            EndpointSnapshot.objects.bulk_create(
                snapshots, 
                ignore_conflicts=True
            )
            
            logger.debug("端点快照保存成功 - 数量: %d", len(snapshots))
            
        except Exception as e:
            logger.error(
                "保存端点快照失败 - 数量: %d, 错误: %s",
                len(items),
                str(e),
                exc_info=True
            )
            raise
    
    def get_by_scan(self, scan_id: int):
        return EndpointSnapshot.objects.filter(scan_id=scan_id).order_by('-created_at')

    def get_all(self):
        return EndpointSnapshot.objects.all().order_by('-created_at')

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
            包含所有端点字段的字典
        """
        qs = (
            EndpointSnapshot.objects
            .filter(scan_id=scan_id)
            .values(
                'url', 'host', 'location', 'title', 'status_code',
                'content_length', 'content_type', 'webserver', 'tech',
                'response_body', 'response_headers', 'vhost', 'matched_gf_patterns', 'created_at'
            )
            .order_by('url')
        )
        
        for row in qs.iterator(chunk_size=batch_size):
            yield row
