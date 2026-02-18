"""智能过滤工具 - 通用查询语法解析和 Django ORM 查询构建

支持的语法：
- field="value"     模糊匹配（包含）
- field=="value"    精确匹配
- field!="value"    不等于

逻辑运算符：
- AND: && 或 and 或 空格（默认）
- OR:  || 或 or

示例：
    type="xss" || type="sqli"           # OR
    type="xss" or type="sqli"           # OR（等价）
    severity="high" && source="nuclei"  # AND
    severity="high" source="nuclei"     # AND（空格默认为 AND）
    severity="high" and source="nuclei" # AND（等价）

使用示例：
    from apps.common.utils.filter_utils import apply_filters
    
    field_mapping = {'ip': 'ip', 'port': 'port', 'host': 'host'}
    queryset = apply_filters(queryset, 'ip="192" || port="80"', field_mapping)
"""

import re
import logging
from dataclasses import dataclass
from typing import List, Dict, Optional, Union
from enum import Enum

from django.db.models import QuerySet, Q, F, Func, CharField
from django.db.models.functions import Cast

logger = logging.getLogger(__name__)


class ArrayToString(Func):
    """PostgreSQL array_to_string 函数"""
    function = 'array_to_string'
    template = "%(function)s(%(expressions)s, ',')"
    output_field = CharField()


class LogicalOp(Enum):
    """逻辑运算符"""
    AND = 'AND'
    OR = 'OR'


@dataclass
class ParsedFilter:
    """解析后的过滤条件"""
    field: str      # 字段名
    operator: str   # 操作符: '=', '==', '!='
    value: str      # 原始值


@dataclass
class FilterGroup:
    """过滤条件组（带逻辑运算符）"""
    filter: ParsedFilter
    logical_op: LogicalOp  # 与前一个条件的逻辑关系


class QueryParser:
    """查询语法解析器
    
    支持 ||/or (OR) 和 &&/and/空格 (AND) 逻辑运算符
    """
    
    # 正则匹配: field="value", field=="value", field!="value"
    FILTER_PATTERN = re.compile(r'(\w+)(==|!=|=)"([^"]*)"')
    
    # 逻辑运算符模式（带空格）
    OR_PATTERN = re.compile(r'\s*(\|\||(?<![a-zA-Z])or(?![a-zA-Z]))\s*', re.IGNORECASE)
    AND_PATTERN = re.compile(r'\s*(&&|(?<![a-zA-Z])and(?![a-zA-Z]))\s*', re.IGNORECASE)
    
    @classmethod
    def parse(cls, query_string: str) -> List[FilterGroup]:
        """解析查询语法字符串
        
        Args:
            query_string: 查询语法字符串
        
        Returns:
            解析后的过滤条件组列表
        
        Examples:
            >>> QueryParser.parse('type="xss" || type="sqli"')
            [FilterGroup(filter=..., logical_op=AND),  # 第一个默认 AND
             FilterGroup(filter=..., logical_op=OR)]
        """
        if not query_string or not query_string.strip():
            return []
        
        # 第一步：提取所有过滤条件并用占位符替换，保护引号内的空格
        filters_found = []
        placeholder_pattern = '__FILTER_{}__'
        
        def replace_filter(match):
            idx = len(filters_found)
            filters_found.append(match.group(0))
            return placeholder_pattern.format(idx)
        
        # 先用正则提取所有 field="value" 形式的条件
        protected = cls.FILTER_PATTERN.sub(replace_filter, query_string)
        
        # 标准化逻辑运算符
        # 先处理 || 和 or -> __OR__
        normalized = cls.OR_PATTERN.sub(' __OR__ ', protected)
        # 再处理 && 和 and -> __AND__
        normalized = cls.AND_PATTERN.sub(' __AND__ ', normalized)
        
        # 分词：按空格分割，保留逻辑运算符标记
        tokens = normalized.split()
        
        groups = []
        pending_op = LogicalOp.AND  # 默认 AND
        
        for token in tokens:
            if token == '__OR__':
                pending_op = LogicalOp.OR
            elif token == '__AND__':
                pending_op = LogicalOp.AND
            elif token.startswith('__FILTER_') and token.endswith('__'):
                # 还原占位符为原始过滤条件
                try:
                    idx = int(token[9:-2])  # 提取索引
                    original_filter = filters_found[idx]
                    match = cls.FILTER_PATTERN.match(original_filter)
                    if match:
                        field, operator, value = match.groups()
                        groups.append(FilterGroup(
                            filter=ParsedFilter(
                                field=field.lower(),
                                operator=operator,
                                value=value
                            ),
                            logical_op=pending_op if groups else LogicalOp.AND
                        ))
                        pending_op = LogicalOp.AND  # 重置为默认 AND
                except (ValueError, IndexError):
                    pass
            # 其他 token 忽略（无效输入）
        
        return groups


class QueryBuilder:
    """Django ORM 查询构建器
    
    将解析后的过滤条件转换为 Django ORM 查询，支持 AND/OR 逻辑
    """
    
    @classmethod
    def build_query(
        cls,
        queryset: QuerySet,
        filter_groups: List[FilterGroup],
        field_mapping: Dict[str, str],
        json_array_fields: List[str] = None
    ) -> QuerySet:
        """构建 Django ORM 查询
        
        Args:
            queryset: Django QuerySet
            filter_groups: 解析后的过滤条件组列表
            field_mapping: 字段映射
            json_array_fields: JSON 数组字段列表（使用 __contains 查询）
        
        Returns:
            过滤后的 QuerySet
        """
        if not filter_groups:
            return queryset
        
        json_array_fields = json_array_fields or []
        
        # 收集需要 annotate 的数组模糊搜索字段
        array_fuzzy_fields = set()
        
        # 第一遍：检查是否有数组模糊匹配
        for group in filter_groups:
            f = group.filter
            db_field = field_mapping.get(f.field)
            if db_field and db_field in json_array_fields and f.operator == '=':
                array_fuzzy_fields.add(db_field)
        
        # 对数组模糊搜索字段做 annotate
        for field in array_fuzzy_fields:
            annotate_name = f'{field}_text'
            queryset = queryset.annotate(**{annotate_name: ArrayToString(F(field))})
        
        # 构建 Q 对象
        combined_q = None
        
        for group in filter_groups:
            f = group.filter
            
            # 字段映射
            db_field = field_mapping.get(f.field)
            if not db_field:
                logger.debug(f"忽略未知字段: {f.field}")
                continue
            
            # 判断是否为 JSON 数组字段
            is_json_array = db_field in json_array_fields
            
            # 构建单个条件的 Q 对象
            q = cls._build_single_q(db_field, f.operator, f.value, is_json_array)
            if q is None:
                continue
            
            # 组合 Q 对象
            if combined_q is None:
                combined_q = q
            elif group.logical_op == LogicalOp.OR:
                combined_q = combined_q | q
            else:  # AND
                combined_q = combined_q & q
        
        if combined_q is not None:
            return queryset.filter(combined_q)
        return queryset
    
    @classmethod
    def _build_single_q(cls, field: str, operator: str, value: str, is_json_array: bool = False) -> Optional[Q]:
        """构建单个条件的 Q 对象"""
        if is_json_array:
            if operator == '==':
                # 精确匹配：数组中包含完全等于 value 的元素
                return Q(**{f'{field}__contains': [value]})
            elif operator == '!=':
                # 不包含：数组中不包含完全等于 value 的元素
                return ~Q(**{f'{field}__contains': [value]})
            else:  # '=' 模糊匹配
                # 使用 annotate 后的字段进行模糊搜索
                # 字段已在 build_query 中通过 ArrayToString 转换为文本
                annotate_name = f'{field}_text'
                return Q(**{f'{annotate_name}__icontains': value})
        
        if operator == '!=':
            return cls._build_not_equal_q(field, value)
        elif operator == '==':
            return cls._build_exact_q(field, value)
        else:  # '='
            return cls._build_fuzzy_q(field, value)
    
    @classmethod
    def _try_convert_to_int(cls, value: str) -> Optional[int]:
        """尝试将值转换为整数"""
        try:
            return int(value.strip())
        except (ValueError, TypeError):
            return None
    
    @classmethod
    def _build_fuzzy_q(cls, field: str, value: str) -> Q:
        """模糊匹配: 包含"""
        return Q(**{f'{field}__icontains': value})
    
    @classmethod
    def _build_exact_q(cls, field: str, value: str) -> Q:
        """精确匹配"""
        int_val = cls._try_convert_to_int(value)
        if int_val is not None:
            return Q(**{f'{field}__exact': int_val})
        return Q(**{f'{field}__exact': value})
    
    @classmethod
    def _build_not_equal_q(cls, field: str, value: str) -> Q:
        """不等于"""
        int_val = cls._try_convert_to_int(value)
        if int_val is not None:
            return ~Q(**{f'{field}__exact': int_val})
        return ~Q(**{f'{field}__exact': value})


def apply_filters(
    queryset: QuerySet,
    query_string: str,
    field_mapping: Dict[str, str],
    json_array_fields: List[str] = None
) -> QuerySet:
    """应用过滤条件到 QuerySet
    
    Args:
        queryset: Django QuerySet
        query_string: 查询语法字符串
        field_mapping: 字段映射
        json_array_fields: JSON 数组字段列表（使用 __contains 查询）
    
    Returns:
        过滤后的 QuerySet
    
    Examples:
        # OR 查询
        apply_filters(qs, 'type="xss" || type="sqli"', mapping)
        apply_filters(qs, 'type="xss" or type="sqli"', mapping)
        
        # AND 查询
        apply_filters(qs, 'severity="high" && source="nuclei"', mapping)
        apply_filters(qs, 'severity="high" source="nuclei"', mapping)
        
        # 混合查询
        apply_filters(qs, 'type="xss" || type="sqli" && severity="high"', mapping)
        
        # JSON 数组字段查询
        apply_filters(qs, 'implies="PHP"', mapping, json_array_fields=['implies'])
    """
    if not query_string or not query_string.strip():
        return queryset
    
    try:
        filter_groups = QueryParser.parse(query_string)
        if not filter_groups:
            logger.debug(f"未解析到有效过滤条件: {query_string}")
            return queryset
        
        logger.debug(f"解析过滤条件: {filter_groups}")
        return QueryBuilder.build_query(
            queryset, 
            filter_groups, 
            field_mapping,
            json_array_fields=json_array_fields
        )
    
    except Exception as e:
        logger.warning(f"过滤解析错误: {e}, query: {query_string}")
        return queryset  # 静默降级
