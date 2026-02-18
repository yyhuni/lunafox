"""Goby 指纹管理 Service

实现 Goby 格式指纹的校验、转换和导出逻辑
"""

from apps.engine.models import GobyFingerprint
from .base import BaseFingerprintService


class GobyFingerprintService(BaseFingerprintService):
    """Goby 指纹管理服务（继承基类，实现 Goby 特定逻辑）"""
    
    model = GobyFingerprint
    
    def validate_fingerprint(self, item: dict) -> bool:
        """
        校验单条 Goby 指纹
        
        支持两种格式：
        1. 标准格式: {"name": "...", "logic": "...", "rule": [...]}
        2. JSONL 格式: {"product": "...", "rule": "..."}
        
        Args:
            item: 单条指纹数据
            
        Returns:
            bool: 是否有效
        """
        # 标准格式：name + logic + rule(数组)
        name = item.get('name', '')
        if name and item.get('logic') is not None and isinstance(item.get('rule'), list):
            return bool(str(name).strip())
        
        # JSONL 格式：product + rule(字符串)
        product = item.get('product', '')
        rule = item.get('rule', '')
        return bool(product and str(product).strip() and rule and str(rule).strip())
    
    def to_model_data(self, item: dict) -> dict:
        """
        转换 Goby JSON 格式为 Model 字段
        
        支持两种输入格式：
        1. 标准格式: {"name": "...", "logic": "...", "rule": [...]}
        2. JSONL 格式: {"product": "...", "rule": "..."}
        
        Args:
            item: 原始 Goby JSON 数据
            
        Returns:
            dict: Model 字段数据
        """
        # 标准格式
        if 'name' in item and isinstance(item.get('rule'), list):
            return {
                'name': str(item.get('name', '')).strip(),
                'logic': item.get('logic', ''),
                'rule': item.get('rule', []),
            }
        
        # JSONL 格式：将 rule 字符串转为单元素数组
        return {
            'name': str(item.get('product', '')).strip(),
            'logic': 'or',  # JSONL 格式默认 or 逻辑
            'rule': [item.get('rule', '')] if item.get('rule') else [],
        }
    
    def get_export_data(self) -> list:
        """
        获取导出数据（Goby JSON 格式 - 数组）
        
        Returns:
            list: Goby 格式的 JSON 数据（数组格式）
            [
                {"name": "...", "logic": "...", "rule": [...]},
                ...
            ]
        """
        fingerprints = self.model.objects.all()
        return [
            {
                'name': fp.name,
                'logic': fp.logic,
                'rule': fp.rule,
            }
            for fp in fingerprints
        ]
