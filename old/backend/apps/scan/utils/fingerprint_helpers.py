"""指纹文件本地缓存工具

提供 Worker 侧的指纹文件缓存和版本校验功能，用于：
- 指纹识别扫描 (fingerprint_detect_flow)
"""

import json
import logging
import os

from django.conf import settings

logger = logging.getLogger(__name__)


# 指纹库映射：lib_name → ensure_func_name
FINGERPRINT_LIB_MAP = {
    'ehole': 'ensure_ehole_fingerprint_local',
    'goby': 'ensure_goby_fingerprint_local',
    'wappalyzer': 'ensure_wappalyzer_fingerprint_local',
    'fingers': 'ensure_fingers_fingerprint_local',
    'fingerprinthub': 'ensure_fingerprinthub_fingerprint_local',
    'arl': 'ensure_arl_fingerprint_local',
}


def ensure_ehole_fingerprint_local() -> str:
    """
    确保本地存在最新的 EHole 指纹文件（带缓存）
    
    流程：
    1. 获取当前指纹库版本
    2. 检查缓存文件是否存在且版本匹配
    3. 版本不匹配则重新导出
    
    Returns:
        str: 本地指纹文件路径
    
    使用场景：
        Worker 执行扫描任务前调用，获取最新指纹文件路径
    """
    from apps.engine.services.fingerprints import EholeFingerprintService
    
    service = EholeFingerprintService()
    current_version = service.get_fingerprint_version()
    
    # 缓存目录和文件
    base_dir = getattr(settings, 'FINGERPRINTS_BASE_PATH', '/opt/xingrin/fingerprints')
    os.makedirs(base_dir, exist_ok=True)
    cache_file = os.path.join(base_dir, 'ehole.json')
    version_file = os.path.join(base_dir, 'ehole.version')
    
    # 检查缓存版本
    cached_version = None
    if os.path.exists(version_file):
        try:
            with open(version_file, 'r') as f:
                cached_version = f.read().strip()
        except OSError as e:
            logger.warning("读取版本文件失败: %s", e)
    
    # 版本匹配，直接返回缓存
    if cached_version == current_version and os.path.exists(cache_file):
        logger.info("EHole 指纹文件缓存有效（版本匹配）: %s", cache_file)
        return cache_file
    
    # 版本不匹配，重新导出
    logger.info(
        "EHole 指纹文件需要更新: cached=%s, current=%s",
        cached_version, current_version
    )
    count = service.export_to_file(cache_file)
    
    # 写入版本文件
    try:
        with open(version_file, 'w') as f:
            f.write(current_version)
    except OSError as e:
        logger.warning("写入版本文件失败: %s", e)
    
    logger.info("EHole 指纹文件已更新: %s", cache_file)
    return cache_file


def ensure_goby_fingerprint_local() -> str:
    """
    确保本地存在最新的 Goby 指纹文件（带缓存）
    
    Returns:
        str: 本地指纹文件路径
    """
    from apps.engine.services.fingerprints import GobyFingerprintService
    
    service = GobyFingerprintService()
    current_version = service.get_fingerprint_version()
    
    # 缓存目录和文件
    base_dir = getattr(settings, 'FINGERPRINTS_BASE_PATH', '/opt/xingrin/fingerprints')
    os.makedirs(base_dir, exist_ok=True)
    cache_file = os.path.join(base_dir, 'goby.json')
    version_file = os.path.join(base_dir, 'goby.version')
    
    # 检查缓存版本
    cached_version = None
    if os.path.exists(version_file):
        try:
            with open(version_file, 'r') as f:
                cached_version = f.read().strip()
        except OSError as e:
            logger.warning("读取 Goby 版本文件失败: %s", e)
    
    # 版本匹配，直接返回缓存
    if cached_version == current_version and os.path.exists(cache_file):
        logger.info("Goby 指纹文件缓存有效（版本匹配）: %s", cache_file)
        return cache_file
    
    # 版本不匹配，重新导出
    logger.info(
        "Goby 指纹文件需要更新: cached=%s, current=%s",
        cached_version, current_version
    )
    # Goby 导出格式是数组，直接写入
    data = service.get_export_data()
    with open(cache_file, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False)
    
    # 写入版本文件
    try:
        with open(version_file, 'w') as f:
            f.write(current_version)
    except OSError as e:
        logger.warning("写入 Goby 版本文件失败: %s", e)
    
    logger.info("Goby 指纹文件已更新: %s", cache_file)
    return cache_file


def ensure_wappalyzer_fingerprint_local() -> str:
    """
    确保本地存在最新的 Wappalyzer 指纹文件（带缓存）
    
    Returns:
        str: 本地指纹文件路径
    """
    from apps.engine.services.fingerprints import WappalyzerFingerprintService
    
    service = WappalyzerFingerprintService()
    current_version = service.get_fingerprint_version()
    
    # 缓存目录和文件
    base_dir = getattr(settings, 'FINGERPRINTS_BASE_PATH', '/opt/xingrin/fingerprints')
    os.makedirs(base_dir, exist_ok=True)
    cache_file = os.path.join(base_dir, 'wappalyzer.json')
    version_file = os.path.join(base_dir, 'wappalyzer.version')
    
    # 检查缓存版本
    cached_version = None
    if os.path.exists(version_file):
        try:
            with open(version_file, 'r') as f:
                cached_version = f.read().strip()
        except OSError as e:
            logger.warning("读取 Wappalyzer 版本文件失败: %s", e)
    
    # 版本匹配，直接返回缓存
    if cached_version == current_version and os.path.exists(cache_file):
        logger.info("Wappalyzer 指纹文件缓存有效（版本匹配）: %s", cache_file)
        return cache_file
    
    # 版本不匹配，重新导出
    logger.info(
        "Wappalyzer 指纹文件需要更新: cached=%s, current=%s",
        cached_version, current_version
    )
    # Wappalyzer 导出格式是 {"apps": {...}}
    data = service.get_export_data()
    with open(cache_file, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False)
    
    # 写入版本文件
    try:
        with open(version_file, 'w') as f:
            f.write(current_version)
    except OSError as e:
        logger.warning("写入 Wappalyzer 版本文件失败: %s", e)
    
    logger.info("Wappalyzer 指纹文件已更新: %s", cache_file)
    return cache_file


def get_fingerprint_paths(lib_names: list) -> dict:
    """
    获取多个指纹库的本地路径
    
    Args:
        lib_names: 指纹库名称列表，如 ['ehole', 'goby']
        
    Returns:
        dict: {lib_name: local_path}，如 {'ehole': '/opt/xingrin/fingerprints/ehole.json'}
        
    示例：
        paths = get_fingerprint_paths(['ehole'])
        # {'ehole': '/opt/xingrin/fingerprints/ehole.json'}
    """
    paths = {}
    for lib_name in lib_names:
        if lib_name not in FINGERPRINT_LIB_MAP:
            logger.warning("不支持的指纹库: %s，跳过", lib_name)
            continue
        
        ensure_func_name = FINGERPRINT_LIB_MAP[lib_name]
        # 获取当前模块中的函数
        ensure_func = globals().get(ensure_func_name)
        if ensure_func is None:
            logger.warning("指纹库 %s 的导出函数 %s 未实现，跳过", lib_name, ensure_func_name)
            continue
        
        try:
            paths[lib_name] = ensure_func()
        except Exception as e:
            logger.error("获取指纹库 %s 路径失败: %s", lib_name, e)
            continue
    
    return paths


def ensure_fingers_fingerprint_local() -> str:
    """
    确保本地存在最新的 Fingers 指纹文件（带缓存）
    
    Returns:
        str: 本地指纹文件路径
    """
    from apps.engine.services.fingerprints import FingersFingerprintService
    
    service = FingersFingerprintService()
    current_version = service.get_fingerprint_version()
    
    # 缓存目录和文件
    base_dir = getattr(settings, 'FINGERPRINTS_BASE_PATH', '/opt/xingrin/fingerprints')
    os.makedirs(base_dir, exist_ok=True)
    cache_file = os.path.join(base_dir, 'fingers.json')
    version_file = os.path.join(base_dir, 'fingers.version')
    
    # 检查缓存版本
    cached_version = None
    if os.path.exists(version_file):
        try:
            with open(version_file, 'r') as f:
                cached_version = f.read().strip()
        except OSError as e:
            logger.warning("读取 Fingers 版本文件失败: %s", e)
    
    # 版本匹配，直接返回缓存
    if cached_version == current_version and os.path.exists(cache_file):
        logger.info("Fingers 指纹文件缓存有效（版本匹配）: %s", cache_file)
        return cache_file
    
    # 版本不匹配，重新导出
    logger.info(
        "Fingers 指纹文件需要更新: cached=%s, current=%s",
        cached_version, current_version
    )
    data = service.get_export_data()
    with open(cache_file, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False)
    
    # 写入版本文件
    try:
        with open(version_file, 'w') as f:
            f.write(current_version)
    except OSError as e:
        logger.warning("写入 Fingers 版本文件失败: %s", e)
    
    logger.info("Fingers 指纹文件已更新: %s", cache_file)
    return cache_file


def ensure_fingerprinthub_fingerprint_local() -> str:
    """
    确保本地存在最新的 FingerPrintHub 指纹文件（带缓存）
    
    Returns:
        str: 本地指纹文件路径
    """
    from apps.engine.services.fingerprints import FingerPrintHubFingerprintService
    
    service = FingerPrintHubFingerprintService()
    current_version = service.get_fingerprint_version()
    
    # 缓存目录和文件
    base_dir = getattr(settings, 'FINGERPRINTS_BASE_PATH', '/opt/xingrin/fingerprints')
    os.makedirs(base_dir, exist_ok=True)
    cache_file = os.path.join(base_dir, 'fingerprinthub.json')
    version_file = os.path.join(base_dir, 'fingerprinthub.version')
    
    # 检查缓存版本
    cached_version = None
    if os.path.exists(version_file):
        try:
            with open(version_file, 'r') as f:
                cached_version = f.read().strip()
        except OSError as e:
            logger.warning("读取 FingerPrintHub 版本文件失败: %s", e)
    
    # 版本匹配，直接返回缓存
    if cached_version == current_version and os.path.exists(cache_file):
        logger.info("FingerPrintHub 指纹文件缓存有效（版本匹配）: %s", cache_file)
        return cache_file
    
    # 版本不匹配，重新导出
    logger.info(
        "FingerPrintHub 指纹文件需要更新: cached=%s, current=%s",
        cached_version, current_version
    )
    data = service.get_export_data()
    with open(cache_file, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False)
    
    # 写入版本文件
    try:
        with open(version_file, 'w') as f:
            f.write(current_version)
    except OSError as e:
        logger.warning("写入 FingerPrintHub 版本文件失败: %s", e)
    
    logger.info("FingerPrintHub 指纹文件已更新: %s", cache_file)
    return cache_file


def ensure_arl_fingerprint_local() -> str:
    """
    确保本地存在最新的 ARL 指纹文件（带缓存）
    
    Returns:
        str: 本地指纹文件路径（YAML 格式）
    """
    import yaml
    from apps.engine.services.fingerprints import ARLFingerprintService
    
    service = ARLFingerprintService()
    current_version = service.get_fingerprint_version()
    
    # 缓存目录和文件
    base_dir = getattr(settings, 'FINGERPRINTS_BASE_PATH', '/opt/xingrin/fingerprints')
    os.makedirs(base_dir, exist_ok=True)
    cache_file = os.path.join(base_dir, 'arl.yaml')
    version_file = os.path.join(base_dir, 'arl.version')
    
    # 检查缓存版本
    cached_version = None
    if os.path.exists(version_file):
        try:
            with open(version_file, 'r') as f:
                cached_version = f.read().strip()
        except OSError as e:
            logger.warning("读取 ARL 版本文件失败: %s", e)
    
    # 版本匹配，直接返回缓存
    if cached_version == current_version and os.path.exists(cache_file):
        logger.info("ARL 指纹文件缓存有效（版本匹配）: %s", cache_file)
        return cache_file
    
    # 版本不匹配，重新导出
    logger.info(
        "ARL 指纹文件需要更新: cached=%s, current=%s",
        cached_version, current_version
    )
    data = service.get_export_data()
    with open(cache_file, 'w', encoding='utf-8') as f:
        yaml.dump(data, f, allow_unicode=True, default_flow_style=False)
    
    # 写入版本文件
    try:
        with open(version_file, 'w') as f:
            f.write(current_version)
    except OSError as e:
        logger.warning("写入 ARL 版本文件失败: %s", e)
    
    logger.info("ARL 指纹文件已更新: %s", cache_file)
    return cache_file


__all__ = [
    "ensure_ehole_fingerprint_local",
    "ensure_goby_fingerprint_local",
    "ensure_wappalyzer_fingerprint_local",
    "ensure_fingers_fingerprint_local",
    "ensure_fingerprinthub_fingerprint_local",
    "ensure_arl_fingerprint_local",
    "get_fingerprint_paths",
    "FINGERPRINT_LIB_MAP",
]
