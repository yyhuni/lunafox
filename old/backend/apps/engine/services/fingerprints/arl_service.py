"""ARL 指纹管理 Service

实现 ARL 格式指纹的校验、转换和导出逻辑
支持 YAML 格式的导入导出
"""

import logging
import yaml

from apps.engine.models import ARLFingerprint
from .base import BaseFingerprintService

logger = logging.getLogger(__name__)


class ARLFingerprintService(BaseFingerprintService):
    """ARL 指纹管理服务（继承基类，实现 ARL 特定逻辑）"""
    
    model = ARLFingerprint
    
    def validate_fingerprint(self, item: dict) -> bool:
        """
        校验单条 ARL 指纹
        
        校验规则：
        - name 字段必须存在且非空
        - rule 字段必须存在且非空
        
        Args:
            item: 单条指纹数据
            
        Returns:
            bool: 是否有效
        """
        name = item.get('name', '')
        rule = item.get('rule', '')
        return bool(name and str(name).strip()) and bool(rule and str(rule).strip())
    
    def to_model_data(self, item: dict) -> dict:
        """
        转换 ARL YAML 格式为 Model 字段
        
        Args:
            item: 原始 ARL YAML 数据
            
        Returns:
            dict: Model 字段数据
        """
        return {
            'name': str(item.get('name', '')).strip(),
            'rule': str(item.get('rule', '')).strip(),
        }
    
    def get_export_data(self) -> list:
        """
        获取导出数据（ARL 格式 - 数组，用于 YAML 导出）
        
        Returns:
            list: ARL 格式的数据（数组格式）
            [
                {"name": "...", "rule": "..."},
                ...
            ]
        """
        fingerprints = self.model.objects.all()
        return [
            {
                'name': fp.name,
                'rule': fp.rule,
            }
            for fp in fingerprints
        ]
    
    def export_to_yaml(self, output_path: str) -> int:
        """
        导出所有指纹到 YAML 文件
        
        Args:
            output_path: 输出文件路径
            
        Returns:
            int: 导出的指纹数量
        """
        data = self.get_export_data()
        with open(output_path, 'w', encoding='utf-8') as f:
            yaml.dump(data, f, allow_unicode=True, default_flow_style=False, sort_keys=False)
        count = len(data)
        logger.info("导出 ARL 指纹文件: %s, 数量: %d", output_path, count)
        return count
    
    def parse_yaml_import(self, yaml_content: str) -> list:
        """
        解析 YAML 格式的导入内容
        
        Args:
            yaml_content: YAML 格式的字符串内容
            
        Returns:
            list: 解析后的指纹数据列表
            
        Raises:
            ValueError: 当 YAML 格式无效时
        """
        try:
            data = yaml.safe_load(yaml_content)
            if not isinstance(data, list):
                raise ValueError("ARL YAML 文件必须是数组格式")
            return data
        except yaml.YAMLError as e:
            raise ValueError(f"无效的 YAML 格式: {e}")
