"""
资产搜索服务

提供资产搜索的核心业务逻辑：
- 从物化视图查询数据
- 支持表达式语法解析
- 支持 =（模糊）、==（精确）、!=（不等于）操作符
- 支持 && (AND) 和 || (OR) 逻辑组合
- 支持 Website 和 Endpoint 两种资产类型
"""

import logging
import re
from typing import Optional, List, Dict, Any, Tuple, Literal, Iterator

from django.db import connection

logger = logging.getLogger(__name__)

# 支持的字段映射（前端字段名 -> 数据库字段名）
FIELD_MAPPING = {
    'host': 'host',
    'url': 'url',
    'title': 'title',
    'tech': 'tech',
    'status': 'status_code',
    'body': 'response_body',
    'header': 'response_headers',
}

# 数组类型字段
ARRAY_FIELDS = {'tech'}

# 资产类型到视图名的映射
VIEW_MAPPING = {
    'website': 'asset_search_view',
    'endpoint': 'endpoint_search_view',
}

# 资产类型到原表名的映射（用于 JOIN 获取数组字段）
# ⚠️ 重要：pg_ivm 不支持 ArrayField，所有数组字段必须从原表 JOIN 获取
TABLE_MAPPING = {
    'website': 'website',
    'endpoint': 'endpoint',
}

# 有效的资产类型
VALID_ASSET_TYPES = {'website', 'endpoint'}

# Website 查询字段（v=视图，t=原表）
# ⚠️ 注意：t.tech 从原表获取，因为 pg_ivm 不支持 ArrayField
WEBSITE_SELECT_FIELDS = """
    v.id,
    v.url,
    v.host,
    v.title,
    t.tech,  -- ArrayField，从 website 表 JOIN 获取
    v.status_code,
    v.response_headers,
    v.response_body,
    v.content_type,
    v.content_length,
    v.webserver,
    v.location,
    v.vhost,
    v.created_at,
    v.target_id
"""

# Endpoint 查询字段
# ⚠️ 注意：t.tech 和 t.matched_gf_patterns 从原表获取，因为 pg_ivm 不支持 ArrayField
ENDPOINT_SELECT_FIELDS = """
    v.id,
    v.url,
    v.host,
    v.title,
    t.tech,  -- ArrayField，从 endpoint 表 JOIN 获取
    v.status_code,
    v.response_headers,
    v.response_body,
    v.content_type,
    v.content_length,
    v.webserver,
    v.location,
    v.vhost,
    t.matched_gf_patterns,  -- ArrayField，从 endpoint 表 JOIN 获取
    v.created_at,
    v.target_id
"""


class SearchQueryParser:
    """
    搜索查询解析器
    
    支持语法：
    - field="value"     模糊匹配（ILIKE %value%）
    - field=="value"    精确匹配
    - field!="value"    不等于
    - &&                AND 连接
    - ||                OR 连接
    - ()                分组（暂不支持嵌套）
    
    示例：
    - host="api" && tech="nginx"
    - tech="vue" || tech="react"
    - status=="200" && host!="test"
    """
    
    # 匹配单个条件: field="value" 或 field=="value" 或 field!="value"
    CONDITION_PATTERN = re.compile(r'(\w+)\s*(==|!=|=)\s*"([^"]*)"')
    
    @classmethod
    def parse(cls, query: str) -> Tuple[str, List[Any]]:
        """
        解析查询字符串，返回 SQL WHERE 子句和参数
        
        Args:
            query: 搜索查询字符串
        
        Returns:
            (where_clause, params) 元组
        """
        if not query or not query.strip():
            return "1=1", []
        
        query = query.strip()
        
        # 检查是否包含操作符语法，如果不包含则作为 host 模糊搜索
        if not cls.CONDITION_PATTERN.search(query):
            # 裸文本，默认作为 host 模糊搜索（v 是视图别名）
            return "v.host ILIKE %s", [f"%{query}%"]
        
        # 按 || 分割为 OR 组
        or_groups = cls._split_by_or(query)
        
        if len(or_groups) == 1:
            # 没有 OR，直接解析 AND 条件
            return cls._parse_and_group(or_groups[0])
        
        # 多个 OR 组
        or_clauses = []
        all_params = []
        
        for group in or_groups:
            clause, params = cls._parse_and_group(group)
            if clause and clause != "1=1":
                or_clauses.append(f"({clause})")
                all_params.extend(params)
        
        if not or_clauses:
            return "1=1", []
        
        return " OR ".join(or_clauses), all_params
    
    @classmethod
    def _split_by_or(cls, query: str) -> List[str]:
        """按 || 分割查询，但忽略引号内的 ||"""
        parts = []
        current = ""
        in_quotes = False
        i = 0
        
        while i < len(query):
            char = query[i]
            
            if char == '"':
                in_quotes = not in_quotes
                current += char
            elif not in_quotes and i + 1 < len(query) and query[i:i+2] == '||':
                if current.strip():
                    parts.append(current.strip())
                current = ""
                i += 1  # 跳过第二个 |
            else:
                current += char
            
            i += 1
        
        if current.strip():
            parts.append(current.strip())
        
        return parts if parts else [query]
    
    @classmethod
    def _parse_and_group(cls, group: str) -> Tuple[str, List[Any]]:
        """解析 AND 组（用 && 连接的条件）"""
        # 移除外层括号
        group = group.strip()
        if group.startswith('(') and group.endswith(')'):
            group = group[1:-1].strip()
        
        # 按 && 分割
        parts = cls._split_by_and(group)
        
        and_clauses = []
        all_params = []
        
        for part in parts:
            clause, params = cls._parse_condition(part.strip())
            if clause:
                and_clauses.append(clause)
                all_params.extend(params)
        
        if not and_clauses:
            return "1=1", []
        
        return " AND ".join(and_clauses), all_params
    
    @classmethod
    def _split_by_and(cls, query: str) -> List[str]:
        """按 && 分割查询，但忽略引号内的 &&"""
        parts = []
        current = ""
        in_quotes = False
        i = 0
        
        while i < len(query):
            char = query[i]
            
            if char == '"':
                in_quotes = not in_quotes
                current += char
            elif not in_quotes and i + 1 < len(query) and query[i:i+2] == '&&':
                if current.strip():
                    parts.append(current.strip())
                current = ""
                i += 1  # 跳过第二个 &
            else:
                current += char
            
            i += 1
        
        if current.strip():
            parts.append(current.strip())
        
        return parts if parts else [query]
    
    @classmethod
    def _parse_condition(cls, condition: str) -> Tuple[Optional[str], List[Any]]:
        """
        解析单个条件
        
        Returns:
            (sql_clause, params) 或 (None, []) 如果解析失败
        """
        # 移除括号
        condition = condition.strip()
        if condition.startswith('(') and condition.endswith(')'):
            condition = condition[1:-1].strip()
        
        match = cls.CONDITION_PATTERN.match(condition)
        if not match:
            logger.warning(f"无法解析条件: {condition}")
            return None, []
        
        field, operator, value = match.groups()
        field = field.lower()
        
        # 验证字段
        if field not in FIELD_MAPPING:
            logger.warning(f"未知字段: {field}")
            return None, []
        
        db_field = FIELD_MAPPING[field]
        is_array = field in ARRAY_FIELDS
        
        # 根据操作符生成 SQL
        if operator == '=':
            # 模糊匹配
            return cls._build_like_condition(db_field, value, is_array)
        elif operator == '==':
            # 精确匹配
            return cls._build_exact_condition(db_field, value, is_array)
        elif operator == '!=':
            # 不等于
            return cls._build_not_equal_condition(db_field, value, is_array)
        
        return None, []
    
    @classmethod
    def _build_like_condition(cls, field: str, value: str, is_array: bool) -> Tuple[str, List[Any]]:
        """构建模糊匹配条件"""
        if is_array:
            # 数组字段：检查数组中是否有元素包含该值（从原表 t 获取）
            return f"EXISTS (SELECT 1 FROM unnest(t.{field}) AS elem WHERE elem ILIKE %s)", [f"%{value}%"]
        elif field == 'status_code':
            # 状态码是整数，模糊匹配转为精确匹配
            try:
                return f"v.{field} = %s", [int(value)]
            except ValueError:
                return f"v.{field}::text ILIKE %s", [f"%{value}%"]
        else:
            return f"v.{field} ILIKE %s", [f"%{value}%"]
    
    @classmethod
    def _build_exact_condition(cls, field: str, value: str, is_array: bool) -> Tuple[str, List[Any]]:
        """构建精确匹配条件"""
        if is_array:
            # 数组字段：检查数组中是否包含该精确值（从原表 t 获取）
            return f"%s = ANY(t.{field})", [value]
        elif field == 'status_code':
            # 状态码是整数
            try:
                return f"v.{field} = %s", [int(value)]
            except ValueError:
                return f"v.{field}::text = %s", [value]
        else:
            return f"v.{field} = %s", [value]
    
    @classmethod
    def _build_not_equal_condition(cls, field: str, value: str, is_array: bool) -> Tuple[str, List[Any]]:
        """构建不等于条件"""
        if is_array:
            # 数组字段：检查数组中不包含该值（从原表 t 获取）
            return f"NOT (%s = ANY(t.{field}))", [value]
        elif field == 'status_code':
            try:
                return f"(v.{field} IS NULL OR v.{field} != %s)", [int(value)]
            except ValueError:
                return f"(v.{field} IS NULL OR v.{field}::text != %s)", [value]
        else:
            return f"(v.{field} IS NULL OR v.{field} != %s)", [value]


AssetType = Literal['website', 'endpoint']


class AssetSearchService:
    """资产搜索服务"""
    
    def search(
        self, 
        query: str, 
        asset_type: AssetType = 'website',
        limit: Optional[int] = None
    ) -> List[Dict[str, Any]]:
        """
        搜索资产
        
        Args:
            query: 搜索查询字符串
            asset_type: 资产类型 ('website' 或 'endpoint')
            limit: 最大返回数量（可选）
        
        Returns:
            List[Dict]: 搜索结果列表
        """
        where_clause, params = SearchQueryParser.parse(query)
        
        # 根据资产类型选择视图、原表和字段
        view_name = VIEW_MAPPING.get(asset_type, 'asset_search_view')
        table_name = TABLE_MAPPING.get(asset_type, 'website')
        select_fields = ENDPOINT_SELECT_FIELDS if asset_type == 'endpoint' else WEBSITE_SELECT_FIELDS
        
        # JOIN 原表获取数组字段（tech, matched_gf_patterns）
        sql = f"""
            SELECT {select_fields}
            FROM {view_name} v
            JOIN {table_name} t ON v.id = t.id
            WHERE {where_clause}
            ORDER BY v.created_at DESC
        """
        
        # 添加 LIMIT
        if limit is not None and limit > 0:
            sql += f" LIMIT {int(limit)}"
        
        try:
            with connection.cursor() as cursor:
                cursor.execute(sql, params)
                columns = [col[0] for col in cursor.description]
                results = []
                
                for row in cursor.fetchall():
                    result = dict(zip(columns, row))
                    results.append(result)
                
                return results
        except Exception as e:
            logger.error(f"搜索查询失败: {e}, SQL: {sql}, params: {params}")
            raise
    
    def count(self, query: str, asset_type: AssetType = 'website', statement_timeout_ms: int = 300000) -> int:
        """
        统计搜索结果数量
        
        Args:
            query: 搜索查询字符串
            asset_type: 资产类型 ('website' 或 'endpoint')
            statement_timeout_ms: SQL 语句超时时间（毫秒），默认 5 分钟
        
        Returns:
            int: 结果总数
        """
        where_clause, params = SearchQueryParser.parse(query)
        
        # 根据资产类型选择视图和原表
        view_name = VIEW_MAPPING.get(asset_type, 'asset_search_view')
        table_name = TABLE_MAPPING.get(asset_type, 'website')
        
        # JOIN 原表以支持数组字段查询
        sql = f"SELECT COUNT(*) FROM {view_name} v JOIN {table_name} t ON v.id = t.id WHERE {where_clause}"
        
        try:
            with connection.cursor() as cursor:
                # 为导出设置更长的超时时间（仅影响当前会话）
                cursor.execute(f"SET LOCAL statement_timeout = {statement_timeout_ms}")
                cursor.execute(sql, params)
                return cursor.fetchone()[0]
        except Exception as e:
            logger.error(f"统计查询失败: {e}")
            raise
    
    def search_iter(
        self, 
        query: str, 
        asset_type: AssetType = 'website',
        batch_size: int = 1000,
        statement_timeout_ms: int = 300000
    ) -> Iterator[Dict[str, Any]]:
        """
        流式搜索资产（使用分批查询，内存友好）
        
        Args:
            query: 搜索查询字符串
            asset_type: 资产类型 ('website' 或 'endpoint')
            batch_size: 每批获取的数量
            statement_timeout_ms: SQL 语句超时时间（毫秒），默认 5 分钟
        
        Yields:
            Dict: 单条搜索结果
        """
        where_clause, params = SearchQueryParser.parse(query)
        
        # 根据资产类型选择视图、原表和字段
        view_name = VIEW_MAPPING.get(asset_type, 'asset_search_view')
        table_name = TABLE_MAPPING.get(asset_type, 'website')
        select_fields = ENDPOINT_SELECT_FIELDS if asset_type == 'endpoint' else WEBSITE_SELECT_FIELDS
        
        # 使用 OFFSET/LIMIT 分批查询（Django 不支持命名游标）
        offset = 0
        
        try:
            while True:
                # JOIN 原表获取数组字段
                sql = f"""
                    SELECT {select_fields}
                    FROM {view_name} v
                    JOIN {table_name} t ON v.id = t.id
                    WHERE {where_clause}
                    ORDER BY v.created_at DESC
                    LIMIT {batch_size} OFFSET {offset}
                """
                
                with connection.cursor() as cursor:
                    # 为导出设置更长的超时时间（仅影响当前会话）
                    cursor.execute(f"SET LOCAL statement_timeout = {statement_timeout_ms}")
                    cursor.execute(sql, params)
                    columns = [col[0] for col in cursor.description]
                    rows = cursor.fetchall()
                
                if not rows:
                    break
                
                for row in rows:
                    yield dict(zip(columns, row))
                
                # 如果返回的行数少于 batch_size，说明已经是最后一批
                if len(rows) < batch_size:
                    break
                
                offset += batch_size
                
        except Exception as e:
            logger.error(f"流式搜索查询失败: {e}, SQL: {sql}, params: {params}")
            raise
