"""
批量数据去重工具

用于 bulk_create 前的批次内去重，避免 PostgreSQL ON CONFLICT 错误。
自动从 Django 模型读取唯一约束字段，无需手动指定。
"""

import logging
from typing import List, TypeVar, Tuple, Optional

from django.db import models

logger = logging.getLogger(__name__)

T = TypeVar('T')


def get_unique_fields(model: type[models.Model]) -> Optional[Tuple[str, ...]]:
    """
    从 Django 模型获取唯一约束字段
    
    按优先级查找：
    1. Meta.constraints 中的 UniqueConstraint
    2. Meta.unique_together
    
    Args:
        model: Django 模型类
        
    Returns:
        唯一约束字段元组，如果没有则返回 None
    """
    meta = model._meta
    
    # 1. 优先查找 UniqueConstraint
    for constraint in getattr(meta, 'constraints', []):
        if isinstance(constraint, models.UniqueConstraint):
            # 跳过条件约束（partial unique）
            if getattr(constraint, 'condition', None) is None:
                return tuple(constraint.fields)
    
    # 2. 回退到 unique_together
    unique_together = getattr(meta, 'unique_together', None)
    if unique_together:
        # unique_together 可能是 (('a', 'b'),) 或 ('a', 'b')
        if unique_together and isinstance(unique_together[0], (list, tuple)):
            return tuple(unique_together[0])
        return tuple(unique_together)
    
    return None


def deduplicate_for_bulk(items: List[T], model: type[models.Model]) -> List[T]:
    """
    根据模型唯一约束对数据去重
    
    自动从模型读取唯一约束字段，生成去重 key。
    保留最后一条记录（后面的数据通常是更新的）。
    
    Args:
        items: 待去重的数据列表（DTO 或 Model 对象）
        model: Django 模型类（用于读取唯一约束）
        
    Returns:
        去重后的数据列表
        
    Example:
        # 自动从 Endpoint 模型读取唯一约束 (url, target)
        unique_items = deduplicate_for_bulk(items, Endpoint)
    """
    if not items:
        return items
    
    unique_fields = get_unique_fields(model)
    if unique_fields is None:
        # 模型没有唯一约束，无需去重
        logger.debug(f"{model.__name__} 没有唯一约束，跳过去重")
        return items
    
    # 处理外键字段名（target -> target_id）
    def make_key(item: T) -> tuple:
        key_parts = []
        for field in unique_fields:
            # 尝试 field_id（外键）和 field 两种形式
            value = getattr(item, f'{field}_id', None)
            if value is None:
                value = getattr(item, field, None)
            key_parts.append(value)
        return tuple(key_parts)
    
    # 使用字典去重，保留最后一条
    seen = {}
    for item in items:
        key = make_key(item)
        seen[key] = item
    
    unique_items = list(seen.values())
    
    if len(unique_items) < len(items):
        logger.debug(f"{model.__name__} 去重: {len(items)} -> {len(unique_items)} 条")
    
    return unique_items
