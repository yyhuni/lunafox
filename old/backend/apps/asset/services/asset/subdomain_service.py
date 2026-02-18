import logging
from typing import List, Dict, Optional
from dataclasses import dataclass

from apps.asset.repositories import DjangoSubdomainRepository
from apps.asset.dtos import SubdomainDTO
from apps.common.validators import is_valid_domain
from apps.common.utils.filter_utils import apply_filters

logger = logging.getLogger(__name__)


@dataclass
class BulkCreateResult:
    """批量创建结果"""
    created_count: int
    skipped_count: int
    invalid_count: int
    mismatched_count: int
    total_received: int


class SubdomainService:
    """子域名业务逻辑层"""
    
    # 智能过滤字段映射
    FILTER_FIELD_MAPPING = {
        'name': 'name',
    }
    
    def __init__(self, repository=None):
        """
        初始化子域名服务
        
        Args:
            repository: 子域名仓储实例（用于依赖注入）
        """
        self.repo = repository or DjangoSubdomainRepository()
    
    # ==================== 查询操作 ====================
    
    def get_all(self, filter_query: Optional[str] = None):
        """
        获取所有子域名
        
        Args:
            filter_query: 智能过滤语法字符串
        
        Returns:
            QuerySet: 子域名查询集
        """
        logger.debug("获取所有子域名")
        queryset = self.repo.get_all()
        if filter_query:
            queryset = apply_filters(queryset, filter_query, self.FILTER_FIELD_MAPPING)
        return queryset
    
    def get_subdomains_by_target(self, target_id: int, filter_query: Optional[str] = None):
        """
        获取目标下的子域名
        
        Args:
            target_id: 目标 ID
            filter_query: 智能过滤语法字符串
        
        Returns:
            QuerySet: 子域名查询集
        """
        queryset = self.repo.get_by_target(target_id)
        if filter_query:
            queryset = apply_filters(queryset, filter_query, self.FILTER_FIELD_MAPPING)
        return queryset

    def count_subdomains_by_target(self, target_id: int) -> int:
        """
        统计目标下的子域名数量
        
        Args:
            target_id: 目标 ID
        
        Returns:
            int: 子域名数量
        """
        logger.debug("统计目标下子域名数量 - Target ID: %d", target_id)
        return self.repo.count_by_target(target_id)
    
    def get_by_names_and_target_id(self, names: set, target_id: int) -> dict:
        """
        根据域名列表和目标ID批量查询子域名
        
        Args:
            names: 域名集合
            target_id: 目标 ID
        
        Returns:
            dict: {域名: Subdomain对象}
        """
        logger.debug("批量查询子域名 - 数量: %d, Target ID: %d", len(names), target_id)
        return self.repo.get_by_names_and_target_id(names, target_id)
    
    def get_subdomain_names_by_target(self, target_id: int) -> List[str]:
        """
        获取目标下的所有子域名名称
        
        Args:
            target_id: 目标 ID
        
        Returns:
            List[str]: 子域名名称列表
        """
        logger.debug("获取目标下所有子域名 - Target ID: %d", target_id)
        return list(self.repo.get_domains_for_export(target_id=target_id))
    
    def iter_subdomain_names_by_target(self, target_id: int, chunk_size: int = 1000):
        """
        流式获取目标下的所有子域名名称（内存优化）
        
        Args:
            target_id: 目标 ID
            chunk_size: 批次大小
        
        Yields:
            str: 子域名名称
        """
        logger.debug("流式获取目标下所有子域名 - Target ID: %d, 批次大小: %d", target_id, chunk_size)
        return self.repo.get_domains_for_export(target_id=target_id, batch_size=chunk_size)

    def iter_raw_data_for_csv_export(self, target_id: int):
        """
        流式获取原始数据用于 CSV 导出
        
        Args:
            target_id: 目标 ID
        
        Yields:
            原始数据字典 {name, created_at}
        """
        return self.repo.iter_raw_data_for_export(target_id=target_id)

    # ==================== 创建操作 ====================

    def bulk_create_ignore_conflicts(self, items: List[SubdomainDTO]) -> None:
        """
        批量创建子域名，忽略冲突
        
        Args:
            items: 子域名 DTO 列表
        
        Note:
            使用 ignore_conflicts 策略，重复记录会被跳过
        """
        logger.debug("批量创建子域名 - 数量: %d", len(items))
        return self.repo.bulk_create_ignore_conflicts(items)

    def bulk_create_subdomains(
        self,
        target_id: int,
        target_name: str,
        subdomains: List[str]
    ) -> BulkCreateResult:
        """
        批量创建子域名（带验证）
        
        Args:
            target_id: 目标 ID
            target_name: 目标域名（用于匹配验证）
            subdomains: 子域名列表
        
        Returns:
            BulkCreateResult: 创建结果统计
        """
        total_received = len(subdomains)
        target_name = target_name.lower().strip()
        
        def is_subdomain_match(subdomain: str) -> bool:
            """验证子域名是否匹配目标域名"""
            if subdomain == target_name:
                return True
            if subdomain.endswith('.' + target_name):
                return True
            return False
        
        # 过滤有效的子域名
        valid_subdomains = []
        invalid_count = 0
        mismatched_count = 0
        
        for subdomain in subdomains:
            if not isinstance(subdomain, str) or not subdomain.strip():
                continue
            
            subdomain = subdomain.lower().strip()
            
            # 验证格式
            if not is_valid_domain(subdomain):
                invalid_count += 1
                continue
            
            # 验证匹配
            if not is_subdomain_match(subdomain):
                mismatched_count += 1
                continue
            
            valid_subdomains.append(subdomain)
        
        # 去重
        unique_subdomains = list(set(valid_subdomains))
        duplicate_count = len(valid_subdomains) - len(unique_subdomains)
        
        if not unique_subdomains:
            return BulkCreateResult(
                created_count=0,
                skipped_count=duplicate_count,
                invalid_count=invalid_count,
                mismatched_count=mismatched_count,
                total_received=total_received,
            )
        
        # 获取创建前的数量
        count_before = self.repo.count_by_target(target_id)
        
        # 创建 DTO 列表并批量创建
        subdomain_dtos = [
            SubdomainDTO(name=name, target_id=target_id)
            for name in unique_subdomains
        ]
        self.repo.bulk_create_ignore_conflicts(subdomain_dtos)
        
        # 获取创建后的数量
        count_after = self.repo.count_by_target(target_id)
        created_count = count_after - count_before
        
        # 计算因数据库冲突跳过的数量
        db_skipped = len(unique_subdomains) - created_count
        
        return BulkCreateResult(
            created_count=created_count,
            skipped_count=duplicate_count + db_skipped,
            invalid_count=invalid_count,
            mismatched_count=mismatched_count,
            total_received=total_received,
        )


__all__ = ['SubdomainService', 'BulkCreateResult']
