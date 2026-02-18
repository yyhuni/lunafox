"""
黑名单规则管理服务

负责黑名单规则的 CRUD 操作（数据库层面）。
过滤逻辑请使用 apps.common.utils.BlacklistFilter。

架构说明：
- Model: BlacklistRule (apps.common.models.blacklist)
- Service: BlacklistService (本文件) - 规则 CRUD
- Utils: BlacklistFilter (apps.common.utils.blacklist_filter) - 过滤逻辑
- View: GlobalBlacklistView, TargetViewSet.blacklist
"""

import logging
from typing import List, Dict, Any, Optional

from django.db.models import QuerySet

from apps.common.utils import detect_rule_type

logger = logging.getLogger(__name__)


def _normalize_patterns(patterns: List[str]) -> List[str]:
    """
    规范化规则列表：去重 + 过滤空行
    
    Args:
        patterns: 原始规则列表
        
    Returns:
        List[str]: 去重后的规则列表（保持顺序）
    """
    return list(dict.fromkeys(filter(None, (p.strip() for p in patterns))))


class BlacklistService:
    """
    黑名单规则管理服务
    
    只负责规则的 CRUD 操作，不包含过滤逻辑。
    过滤逻辑请使用 BlacklistFilter 工具类。
    """
    
    def get_global_rules(self) -> QuerySet:
        """
        获取全局黑名单规则列表
        
        Returns:
            QuerySet: 全局规则查询集
        """
        from apps.common.models import BlacklistRule
        return BlacklistRule.objects.filter(scope=BlacklistRule.Scope.GLOBAL)
    
    def get_target_rules(self, target_id: int) -> QuerySet:
        """
        获取 Target 级黑名单规则列表
        
        Args:
            target_id: Target ID
            
        Returns:
            QuerySet: Target 级规则查询集
        """
        from apps.common.models import BlacklistRule
        return BlacklistRule.objects.filter(
            scope=BlacklistRule.Scope.TARGET,
            target_id=target_id
        )
    
    def get_rules(self, target_id: Optional[int] = None) -> List:
        """
        获取黑名单规则（全局 + Target 级）
        
        Args:
            target_id: Target ID，用于加载 Target 级规则
            
        Returns:
            List[BlacklistRule]: 规则列表
        """
        from apps.common.models import BlacklistRule
        
        # 加载全局规则
        rules = list(BlacklistRule.objects.filter(scope=BlacklistRule.Scope.GLOBAL))
        
        # 加载 Target 级规则
        if target_id:
            target_rules = BlacklistRule.objects.filter(
                scope=BlacklistRule.Scope.TARGET,
                target_id=target_id
            )
            rules.extend(target_rules)
        
        return rules
    
    def replace_global_rules(self, patterns: List[str]) -> Dict[str, Any]:
        """
        全量替换全局黑名单规则（PUT 语义）
        
        Args:
            patterns: 新的规则模式列表
            
        Returns:
            Dict: {'count': int} 最终规则数量
        """
        from apps.common.models import BlacklistRule
        
        count = self._replace_rules(
            patterns=patterns,
            scope=BlacklistRule.Scope.GLOBAL,
            target=None
        )
        
        logger.info("全量替换全局黑名单规则: %d 条", count)
        return {'count': count}
    
    def replace_target_rules(self, target, patterns: List[str]) -> Dict[str, Any]:
        """
        全量替换 Target 级黑名单规则（PUT 语义）
        
        Args:
            target: Target 对象
            patterns: 新的规则模式列表
            
        Returns:
            Dict: {'count': int} 最终规则数量
        """
        from apps.common.models import BlacklistRule
        
        count = self._replace_rules(
            patterns=patterns,
            scope=BlacklistRule.Scope.TARGET,
            target=target
        )
        
        logger.info("全量替换 Target 黑名单规则: %d 条 (Target: %s)", count, target.name)
        return {'count': count}
    
    def _replace_rules(self, patterns: List[str], scope: str, target=None) -> int:
        """
        内部方法：全量替换规则
        
        Args:
            patterns: 规则模式列表
            scope: 规则作用域 (GLOBAL/TARGET)
            target: Target 对象（仅 TARGET 作用域需要）
            
        Returns:
            int: 最终规则数量
        """
        from apps.common.models import BlacklistRule
        from django.db import transaction
        
        patterns = _normalize_patterns(patterns)
        
        with transaction.atomic():
            # 1. 删除旧规则
            delete_filter = {'scope': scope}
            if target:
                delete_filter['target'] = target
            BlacklistRule.objects.filter(**delete_filter).delete()
            
            # 2. 创建新规则
            if patterns:
                rules = [
                    BlacklistRule(
                        pattern=pattern,
                        rule_type=detect_rule_type(pattern),
                        scope=scope,
                        target=target
                    )
                    for pattern in patterns
                ]
                BlacklistRule.objects.bulk_create(rules)
        
        return len(patterns)
