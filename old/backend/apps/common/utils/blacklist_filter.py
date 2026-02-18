"""
黑名单过滤工具

提供域名、IP、CIDR、关键词的黑名单匹配功能。
纯工具类，不涉及数据库操作。

支持的规则类型：
    1. 域名精确匹配: example.com
       - 规则: example.com
       - 匹配: example.com
       - 不匹配: sub.example.com, other.com
    
    2. 域名后缀匹配: *.example.com
       - 规则: *.example.com
       - 匹配: sub.example.com, a.b.example.com, example.com
       - 不匹配: other.com, example.com.cn
    
    3. 关键词匹配: *cdn*
       - 规则: *cdn*
       - 匹配: cdn.example.com, a.cdn.b.com, mycdn123.com
       - 不匹配: example.com (不包含 cdn)
    
    4. IP 精确匹配: 192.168.1.1
       - 规则: 192.168.1.1
       - 匹配: 192.168.1.1
       - 不匹配: 192.168.1.2
    
    5. CIDR 范围匹配: 192.168.0.0/24
       - 规则: 192.168.0.0/24
       - 匹配: 192.168.0.1, 192.168.0.255
       - 不匹配: 192.168.1.1

使用方式：
    from apps.common.utils import BlacklistFilter
    
    # 创建过滤器（传入规则列表）
    rules = BlacklistRule.objects.filter(...)
    filter = BlacklistFilter(rules)
    
    # 检查单个目标
    if filter.is_allowed('http://example.com'):
        process(url)
    
    # 流式处理
    for url in urls:
        if filter.is_allowed(url):
            process(url)
"""

import ipaddress
import logging
from typing import List, Optional
from urllib.parse import urlparse

from apps.common.validators import is_valid_ip, validate_cidr

logger = logging.getLogger(__name__)


def detect_rule_type(pattern: str) -> str:
    """
    自动识别规则类型
    
    支持的模式：
    - 域名精确匹配: example.com
    - 域名后缀匹配: *.example.com
    - 关键词匹配: *cdn* (匹配包含 cdn 的域名)
    - IP 精确匹配: 192.168.1.1
    - CIDR 范围: 192.168.0.0/24
    
    Args:
        pattern: 规则模式字符串
        
    Returns:
        str: 规则类型 ('domain', 'ip', 'cidr', 'keyword')
    """
    if not pattern:
        return 'domain'
    
    pattern = pattern.strip()
    
    # 检查关键词模式: *keyword* (前后都有星号，中间无点)
    if pattern.startswith('*') and pattern.endswith('*') and len(pattern) > 2:
        keyword = pattern[1:-1]
        # 关键词中不能有点（否则可能是域名模式）
        if '.' not in keyword:
            return 'keyword'
    
    # 检查 CIDR（包含 /）
    if '/' in pattern:
        try:
            validate_cidr(pattern)
            return 'cidr'
        except ValueError:
            pass
    
    # 检查 IP（去掉通配符前缀后验证）
    clean_pattern = pattern.lstrip('*').lstrip('.')
    if is_valid_ip(clean_pattern):
        return 'ip'
    
    # 默认为域名
    return 'domain'


def extract_host(target: str) -> str:
    """
    从目标字符串中提取主机名
    
    支持：
    - 纯域名：example.com
    - 纯 IP：192.168.1.1
    - URL：http://example.com/path
    
    Args:
        target: 目标字符串
        
    Returns:
        str: 提取的主机名
    """
    if not target:
        return ''
    
    target = target.strip()
    
    # 如果是 URL，提取 hostname
    if '://' in target:
        try:
            parsed = urlparse(target)
            return parsed.hostname or target
        except Exception:
            return target
    
    return target


class BlacklistFilter:
    """
    黑名单过滤器
    
    预编译规则，提供高效的匹配功能。
    """
    
    def __init__(self, rules: List):
        """
        初始化过滤器
        
        Args:
            rules: BlacklistRule 对象列表
        """
        from apps.common.models import BlacklistRule
        
        # 预解析：按类型分类 + CIDR 预编译
        self._domain_rules = []  # (pattern, is_wildcard, suffix)
        self._ip_rules = set()   # 精确 IP 用 set，O(1) 查找
        self._cidr_rules = []    # (pattern, network_obj)
        self._keyword_rules = [] # 关键词列表（小写）
        
        # 去重：跨 scope 可能有重复规则
        seen_patterns = set()
        
        for rule in rules:
            if rule.pattern in seen_patterns:
                continue
            seen_patterns.add(rule.pattern)
            if rule.rule_type == BlacklistRule.RuleType.DOMAIN:
                pattern = rule.pattern.lower()
                if pattern.startswith('*.'):
                    self._domain_rules.append((pattern, True, pattern[1:]))
                else:
                    self._domain_rules.append((pattern, False, None))
            elif rule.rule_type == BlacklistRule.RuleType.IP:
                self._ip_rules.add(rule.pattern)
            elif rule.rule_type == BlacklistRule.RuleType.CIDR:
                try:
                    network = ipaddress.ip_network(rule.pattern, strict=False)
                    self._cidr_rules.append((rule.pattern, network))
                except ValueError:
                    pass
            elif rule.rule_type == BlacklistRule.RuleType.KEYWORD:
                # *cdn* -> cdn
                keyword = rule.pattern[1:-1].lower()
                self._keyword_rules.append(keyword)
    
    def is_allowed(self, target: str) -> bool:
        """
        检查目标是否通过过滤
        
        Args:
            target: 要检查的目标（域名/IP/URL）
            
        Returns:
            bool: True 表示通过（不在黑名单），False 表示被过滤
        """
        if not target:
            return True
        
        host = extract_host(target)
        if not host:
            return True
        
        # 先判断输入类型，再走对应分支
        if is_valid_ip(host):
            return self._check_ip_rules(host)
        else:
            return self._check_domain_rules(host)
    
    def _check_domain_rules(self, host: str) -> bool:
        """检查域名规则（精确匹配 + 后缀匹配 + 关键词匹配）"""
        host_lower = host.lower()
        
        # 1. 域名规则（精确 + 后缀）
        for pattern, is_wildcard, suffix in self._domain_rules:
            if is_wildcard:
                if host_lower.endswith(suffix) or host_lower == pattern[2:]:
                    return False
            else:
                if host_lower == pattern:
                    return False
        
        # 2. 关键词匹配（字符串 in 操作，O(n*m)）
        for keyword in self._keyword_rules:
            if keyword in host_lower:
                return False
        
        return True
    
    def _check_ip_rules(self, host: str) -> bool:
        """检查 IP 规则（精确匹配 + CIDR）"""
        # 1. IP 精确匹配（O(1)）
        if host in self._ip_rules:
            return False
        
        # 2. CIDR 匹配
        if self._cidr_rules:
            try:
                ip_obj = ipaddress.ip_address(host)
                for _, network in self._cidr_rules:
                    if ip_obj in network:
                        return False
            except ValueError:
                pass
        
        return True
    

