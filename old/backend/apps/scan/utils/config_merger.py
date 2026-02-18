"""
配置合并工具模块

提供多引擎 YAML 配置的冲突检测和合并功能。
"""

from typing import List, Tuple

import yaml


class ConfigConflictError(Exception):
    """配置冲突异常
    
    当两个或多个引擎定义相同的顶层扫描类型键时抛出。
    """
    
    def __init__(self, conflicts: List[Tuple[str, str, str]]):
        """
        参数:
            conflicts: (键, 引擎1名称, 引擎2名称) 元组列表
        """
        self.conflicts = conflicts
        msg = "; ".join([f"{k} 同时存在于「{e1}」和「{e2}」" for k, e1, e2 in conflicts])
        super().__init__(f"扫描类型冲突: {msg}")


def merge_engine_configs(engines: List[Tuple[str, str]]) -> str:
    """
    合并多个引擎的 YAML 配置。
    
    参数:
        engines: (引擎名称, 配置YAML) 元组列表
    
    返回:
        合并后的 YAML 字符串
    
    异常:
        ConfigConflictError: 当顶层键冲突时
    """
    if not engines:
        return ""
    
    if len(engines) == 1:
        return engines[0][1]
    
    # 追踪每个顶层键属于哪个引擎
    key_to_engine: dict[str, str] = {}
    conflicts: List[Tuple[str, str, str]] = []
    
    for engine_name, config_yaml in engines:
        if not config_yaml or not config_yaml.strip():
            continue
            
        try:
            parsed = yaml.safe_load(config_yaml)
        except yaml.YAMLError:
            # 无效 YAML 跳过
            continue
            
        if not isinstance(parsed, dict):
            continue
            
        # 检查顶层键冲突
        for key in parsed.keys():
            if key in key_to_engine:
                conflicts.append((key, key_to_engine[key], engine_name))
            else:
                key_to_engine[key] = engine_name
    
    if conflicts:
        raise ConfigConflictError(conflicts)
    
    # 无冲突，用双换行符连接配置
    configs = []
    for _, config_yaml in engines:
        if config_yaml and config_yaml.strip():
            configs.append(config_yaml.strip())
    
    return "\n\n".join(configs)
