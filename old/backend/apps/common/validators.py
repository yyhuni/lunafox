"""域名、IP、端口、URL 和目标验证工具函数"""
import ipaddress
import logging
from urllib.parse import urlparse

import validators

logger = logging.getLogger(__name__)


def validate_domain(domain: str) -> None:
    """
    验证域名格式（使用 validators 库）
    
    Args:
        domain: 域名字符串（应该已经规范化）
        
    Raises:
        ValueError: 域名格式无效
    """
    if not domain:
        raise ValueError("域名不能为空")
    
    # 使用 validators 库验证域名格式
    # 支持国际化域名（IDN）和各种边界情况
    if not validators.domain(domain):
        raise ValueError(f"域名格式无效: {domain}")


def is_valid_domain(domain: str) -> bool:
    """
    判断是否为有效域名（不抛异常）
    
    Args:
        domain: 域名字符串
        
    Returns:
        bool: 是否为有效域名
    """
    if not domain or len(domain) > 253:
        return False
    return bool(validators.domain(domain))


def validate_ip(ip: str) -> None:
    """
    验证 IP 地址格式（支持 IPv4 和 IPv6）
    
    Args:
        ip: IP 地址字符串（应该已经规范化）
        
    Raises:
        ValueError: IP 地址格式无效
    """
    if not ip:
        raise ValueError("IP 地址不能为空")
    
    try:
        ipaddress.ip_address(ip)
    except ValueError:
        raise ValueError(f"IP 地址格式无效: {ip}")


def is_valid_ip(ip: str) -> bool:
    """
    判断是否为有效 IP 地址（不抛异常）
    
    Args:
        ip: IP 地址字符串
        
    Returns:
        bool: 是否为有效 IP 地址
    """
    if not ip:
        return False
    try:
        ipaddress.ip_address(ip)
        return True
    except ValueError:
        return False


def validate_cidr(cidr: str) -> None:
    """
    验证 CIDR 格式（支持 IPv4 和 IPv6）
    
    Args:
        cidr: CIDR 字符串（应该已经规范化）
        
    Raises:
        ValueError: CIDR 格式无效
    """
    if not cidr:
        raise ValueError("CIDR 不能为空")
    
    try:
        ipaddress.ip_network(cidr, strict=False)
    except ValueError:
        raise ValueError(f"CIDR 格式无效: {cidr}")


def detect_target_type(name: str) -> str:
    """
    检测目标类型（不做规范化，只验证）
    
    Args:
        name: 目标名称（应该已经规范化）
        
    Returns:
        str: 目标类型 ('domain', 'ip', 'cidr') - 使用 Target.TargetType 枚举值
        
    Raises:
        ValueError: 如果无法识别目标类型
    """
    # 在函数内部导入模型，避免 AppRegistryNotReady 错误
    from apps.targets.models import Target

    if not name:
        raise ValueError("目标名称不能为空")
    
    # 检查是否是 CIDR 格式（包含 /）
    if '/' in name:
        validate_cidr(name)
        return Target.TargetType.CIDR
    
    # 检查是否是 IP 地址
    try:
        validate_ip(name)
        return Target.TargetType.IP
    except ValueError:
        pass
    
    # 检查是否是合法域名
    try:
        validate_domain(name)
        return Target.TargetType.DOMAIN
    except ValueError:
        pass
    
    # 无法识别的格式
    raise ValueError(f"无法识别的目标格式: {name}，必须是域名、IP地址或CIDR范围")


def validate_port(port: any) -> tuple[bool, int | None]:
    """
    验证并转换端口号
    
    Args:
        port: 待验证的端口号（可能是字符串、整数或其他类型）
    
    Returns:
        tuple: (is_valid, port_number)
            - is_valid: 端口是否有效
            - port_number: 有效时为整数端口号，无效时为 None
    
    验证规则：
        1. 必须能转换为整数
        2. 必须在 1-65535 范围内
    
    示例：
        >>> is_valid, port_num = validate_port(8080)
        >>> is_valid, port_num
        (True, 8080)
        
        >>> is_valid, port_num = validate_port("invalid")
        >>> is_valid, port_num
        (False, None)
    """
    try:
        port_num = int(port)
        if 1 <= port_num <= 65535:
            return True, port_num
        else:
            logger.warning("端口号超出有效范围 (1-65535): %d", port_num)
            return False, None
    except (ValueError, TypeError):
        logger.warning("端口号格式错误，无法转换为整数: %s", port)
        return False, None


# ==================== URL 验证函数 ====================

def validate_url(url: str) -> None:
    """
    验证 URL 格式，必须包含 scheme（http:// 或 https://）
    
    Args:
        url: URL 字符串
        
    Raises:
        ValueError: URL 格式无效或缺少 scheme
    """
    if not url:
        raise ValueError("URL 不能为空")
    
    # 检查是否包含 scheme
    if not url.startswith('http://') and not url.startswith('https://'):
        raise ValueError("URL 必须包含协议（http:// 或 https://）")
    
    try:
        parsed = urlparse(url)
        if not parsed.hostname:
            raise ValueError("URL 必须包含主机名")
    except Exception:
        raise ValueError(f"URL 格式无效: {url}")


def is_valid_url(url: str, max_length: int = 2000) -> bool:
    """
    判断是否为有效 URL（不抛异常）
    
    Args:
        url: URL 字符串
        max_length: URL 最大长度，默认 2000
        
    Returns:
        bool: 是否为有效 URL
    """
    if not url or len(url) > max_length:
        return False
    try:
        validate_url(url)
        return True
    except ValueError:
        return False


def is_url_match_target(url: str, target_name: str, target_type: str) -> bool:
    """
    判断 URL 是否匹配目标
    
    Args:
        url: URL 字符串
        target_name: 目标名称（域名、IP 或 CIDR）
        target_type: 目标类型 ('domain', 'ip', 'cidr')
        
    Returns:
        bool: 是否匹配
    """
    try:
        parsed = urlparse(url)
        hostname = parsed.hostname
        if not hostname:
            return False
        
        hostname = hostname.lower()
        target_name = target_name.lower()
        
        if target_type == 'domain':
            # 域名类型：hostname 等于 target_name 或以 .target_name 结尾
            return hostname == target_name or hostname.endswith('.' + target_name)
        
        elif target_type == 'ip':
            # IP 类型：hostname 必须完全等于 target_name
            return hostname == target_name
        
        elif target_type == 'cidr':
            # CIDR 类型：hostname 必须是 IP 且在 CIDR 范围内
            try:
                ip = ipaddress.ip_address(hostname)
                network = ipaddress.ip_network(target_name, strict=False)
                return ip in network
            except ValueError:
                # hostname 不是有效 IP
                return False
        
        return False
    except Exception:
        return False


def detect_input_type(input_str: str) -> str:
    """
    检测输入类型（用于快速扫描输入解析）
    
    Args:
        input_str: 输入字符串（应该已经 strip）
        
    Returns:
        str: 输入类型 ('url', 'domain', 'ip', 'cidr')
    """
    if not input_str:
        raise ValueError("输入不能为空")
    
    # 1. 包含 :// 一定是 URL
    if '://' in input_str:
        return 'url'
    
    # 2. 包含 / 需要判断是 CIDR 还是 URL（缺少 scheme）
    if '/' in input_str:
        # CIDR 格式: IP/prefix，如 10.0.0.0/8
        parts = input_str.split('/')
        if len(parts) == 2:
            ip_part, prefix_part = parts
            # 如果斜杠后是纯数字且在 0-32 范围内，检查是否是 CIDR
            if prefix_part.isdigit() and 0 <= int(prefix_part) <= 32:
                ip_parts = ip_part.split('.')
                if len(ip_parts) == 4 and all(p.isdigit() for p in ip_parts):
                    return 'cidr'
        # 不是 CIDR，视为 URL（缺少 scheme，后续验证会报错）
        return 'url'
    
    # 3. 检查是否是 IP 地址
    try:
        ipaddress.ip_address(input_str)
        return 'ip'
    except ValueError:
        pass
    
    # 4. 默认为域名
    return 'domain'
