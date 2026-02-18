"""HostPortMapping Service - 业务逻辑层"""

import logging
from typing import List, Iterator, Optional, Dict

from django.db.models import Min

from apps.asset.repositories.asset import DjangoHostPortMappingRepository
from apps.asset.dtos.asset import HostPortMappingDTO
from apps.common.utils.filter_utils import apply_filters

logger = logging.getLogger(__name__)


class HostPortMappingService:
    """主机端口映射服务 - 负责主机端口映射数据的业务逻辑
    
    职责：
    - 业务逻辑处理（过滤、聚合）
    - 调用 Repository 进行数据访问
    """
    
    # 智能过滤字段映射
    FILTER_FIELD_MAPPING = {
        'ip': 'ip',
        'port': 'port',
        'host': 'host',
    }
    
    def __init__(self):
        self.repo = DjangoHostPortMappingRepository()
    
    def bulk_create_ignore_conflicts(self, items: List[HostPortMappingDTO]) -> int:
        """
        批量创建主机端口映射（忽略冲突）
        
        Args:
            items: 主机端口映射 DTO 列表
        
        Returns:
            int: 实际创建的记录数
        
        Note:
            使用数据库唯一约束 + ignore_conflicts 自动去重
        """
        try:
            logger.debug("Service: 准备批量创建主机端口映射 - 数量: %d", len(items))
            
            created_count = self.repo.bulk_create_ignore_conflicts(items)
            
            logger.info("Service: 主机端口映射创建成功 - 数量: %d", created_count)
            
            return created_count
            
        except Exception as e:
            logger.error(
                "Service: 批量创建主机端口映射失败 - 数量: %d, 错误: %s",
                len(items),
                str(e),
                exc_info=True
            )
            raise

    def iter_host_port_by_target(self, target_id: int, batch_size: int = 1000):
        return self.repo.get_for_export(target_id=target_id, batch_size=batch_size)

    def get_ip_aggregation_by_target(
        self, 
        target_id: int, 
        filter_query: Optional[str] = None
    ) -> List[Dict]:
        """获取目标下的 IP 聚合数据
        
        Args:
            target_id: 目标 ID
            filter_query: 智能过滤语法字符串
        
        Returns:
            聚合后的 IP 数据列表
        """
        # 从 Repository 获取基础 QuerySet
        qs = self.repo.get_queryset_by_target(target_id)
        
        # Service 层应用过滤逻辑
        if filter_query:
            qs = apply_filters(qs, filter_query, self.FILTER_FIELD_MAPPING)
        
        # Service 层处理聚合逻辑
        return self._aggregate_by_ip(qs, filter_query, target_id=target_id)

    def get_all_ip_aggregation(self, filter_query: Optional[str] = None) -> List[Dict]:
        """获取所有 IP 聚合数据（全局查询）
        
        Args:
            filter_query: 智能过滤语法字符串
        
        Returns:
            聚合后的 IP 数据列表
        """
        # 从 Repository 获取基础 QuerySet
        qs = self.repo.get_all_queryset()
        
        # Service 层应用过滤逻辑
        if filter_query:
            qs = apply_filters(qs, filter_query, self.FILTER_FIELD_MAPPING)
        
        # Service 层处理聚合逻辑
        return self._aggregate_by_ip(qs, filter_query)

    def _aggregate_by_ip(
        self, 
        qs, 
        filter_query: Optional[str] = None,
        target_id: Optional[int] = None
    ) -> List[Dict]:
        """按 IP 聚合数据
        
        Args:
            qs: 已过滤的 QuerySet
            filter_query: 过滤条件（用于子查询）
            target_id: 目标 ID（用于子查询限定范围）
        
        Returns:
            聚合后的数据列表
        """
        ip_aggregated = (
            qs
            .values('ip')
            .annotate(created_at=Min('created_at'))
            .order_by('-created_at')
        )

        results = []
        for item in ip_aggregated:
            ip = item['ip']
            
            # 获取该 IP 的所有 host 和 port（也需要应用过滤条件）
            mappings_qs = self.repo.get_queryset_by_ip(ip, target_id=target_id)
            if filter_query:
                mappings_qs = apply_filters(mappings_qs, filter_query, self.FILTER_FIELD_MAPPING)
            
            mappings = mappings_qs.values('host', 'port').distinct()
            hosts = sorted({m['host'] for m in mappings})
            ports = sorted({m['port'] for m in mappings})
            
            results.append({
                'ip': ip,
                'hosts': hosts,
                'ports': ports,
                'created_at': item['created_at'],
            })
        
        return results

    def iter_ips_by_target(self, target_id: int, batch_size: int = 1000) -> Iterator[str]:
        """流式获取目标下的所有唯一 IP 地址。"""
        return self.repo.get_ips_for_export(target_id=target_id, batch_size=batch_size)

    def iter_raw_data_for_csv_export(self, target_id: int) -> Iterator[dict]:
        """
        流式获取原始数据用于 CSV 导出
        
        Args:
            target_id: 目标 ID
        
        Yields:
            原始数据字典 {ip, host, port, created_at}
        """
        return self.repo.iter_raw_data_for_export(target_id=target_id)
