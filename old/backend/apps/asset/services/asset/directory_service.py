"""Directory Service - 目录业务逻辑层"""

import logging
from typing import List, Iterator, Optional

from apps.asset.repositories import DjangoDirectoryRepository
from apps.asset.dtos import DirectoryDTO
from apps.common.validators import is_valid_url, is_url_match_target
from apps.common.utils.filter_utils import apply_filters

logger = logging.getLogger(__name__)


class DirectoryService:
    """目录业务逻辑层"""
    
    # 智能过滤字段映射
    FILTER_FIELD_MAPPING = {
        'url': 'url',
        'status': 'status',
    }
    
    def __init__(self, repository=None):
        """初始化目录服务"""
        self.repo = repository or DjangoDirectoryRepository()
    
    def bulk_upsert(self, directory_dtos: List[DirectoryDTO]) -> int:
        """
        批量创建或更新目录（upsert）
        
        存在则更新所有字段，不存在则创建。
        
        Args:
            directory_dtos: DirectoryDTO 列表
            
        Returns:
            int: 处理的记录数
        """
        if not directory_dtos:
            return 0
        
        try:
            return self.repo.bulk_upsert(directory_dtos)
        except Exception as e:
            logger.error(f"批量 upsert 目录失败: {e}")
            raise
    
    def bulk_create_urls(self, target_id: int, target_name: str, target_type: str, urls: List[str]) -> int:
        """
        批量创建目录（仅 URL，使用 ignore_conflicts）
        
        验证 URL 格式和匹配，过滤无效/不匹配 URL，去重后批量创建。
        已存在的记录会被跳过。
        
        Args:
            target_id: 目标 ID
            target_name: 目标名称（用于匹配验证）
            target_type: 目标类型 ('domain', 'ip', 'cidr')
            urls: URL 列表
            
        Returns:
            int: 实际创建的记录数
        """
        if not urls:
            return 0
        
        # 过滤有效 URL 并去重
        valid_urls = []
        seen = set()
        
        for url in urls:
            if not isinstance(url, str):
                continue
            url = url.strip()
            if not url or url in seen:
                continue
            if not is_valid_url(url):
                continue
            
            # 匹配验证（前端已阻止不匹配的提交，后端作为双重保障）
            if not is_url_match_target(url, target_name, target_type):
                continue
            
            seen.add(url)
            valid_urls.append(url)
        
        if not valid_urls:
            return 0
        
        # 获取创建前的数量
        count_before = self.repo.count_by_target(target_id)
        
        # 创建 DTO 列表并批量创建
        directory_dtos = [
            DirectoryDTO(url=url, target_id=target_id)
            for url in valid_urls
        ]
        self.repo.bulk_create_ignore_conflicts(directory_dtos)
        
        # 获取创建后的数量
        count_after = self.repo.count_by_target(target_id)
        return count_after - count_before
    
    def get_directories_by_target(self, target_id: int, filter_query: Optional[str] = None):
        """获取目标下的所有目录"""
        queryset = self.repo.get_by_target(target_id)
        if filter_query:
            queryset = apply_filters(queryset, filter_query, self.FILTER_FIELD_MAPPING)
        return queryset
    
    def get_all(self, filter_query: Optional[str] = None):
        """获取所有目录"""
        queryset = self.repo.get_all()
        if filter_query:
            queryset = apply_filters(queryset, filter_query, self.FILTER_FIELD_MAPPING)
        return queryset

    def iter_directory_urls_by_target(self, target_id: int, chunk_size: int = 1000) -> Iterator[str]:
        """流式获取目标下的所有目录 URL"""
        return self.repo.get_urls_for_export(target_id=target_id, batch_size=chunk_size)

    def iter_raw_data_for_csv_export(self, target_id: int) -> Iterator[dict]:
        """
        流式获取原始数据用于 CSV 导出
        
        Args:
            target_id: 目标 ID
        
        Yields:
            原始数据字典
        """
        return self.repo.iter_raw_data_for_export(target_id=target_id)


__all__ = ['DirectoryService']
