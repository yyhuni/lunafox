"""Fingers 指纹管理 Service

实现 Fingers 格式指纹的校验、转换和导出逻辑
"""

from apps.engine.models import FingersFingerprint
from .base import BaseFingerprintService


class FingersFingerprintService(BaseFingerprintService):
    """Fingers 指纹管理服务（继承基类，实现 Fingers 特定逻辑）"""
    
    model = FingersFingerprint
    
    def validate_fingerprint(self, item: dict) -> bool:
        """
        校验单条 Fingers 指纹
        
        校验规则：
        - name 字段必须存在且非空
        - rule 字段必须是数组
        
        Args:
            item: 单条指纹数据
            
        Returns:
            bool: 是否有效
        """
        name = item.get('name', '')
        rule = item.get('rule')
        return bool(name and str(name).strip()) and isinstance(rule, list)
    
    def to_model_data(self, item: dict) -> dict:
        """
        转换 Fingers JSON 格式为 Model 字段
        
        字段映射：
        - default_port (JSON) → default_port (Model)
        
        Args:
            item: 原始 Fingers JSON 数据
            
        Returns:
            dict: Model 字段数据
        """
        return {
            'name': str(item.get('name', '')).strip(),
            'link': item.get('link', ''),
            'rule': item.get('rule', []),
            'tag': item.get('tag', []),
            'focus': item.get('focus', False),
            'default_port': item.get('default_port', []),
        }
    
    def get_export_data(self) -> list:
        """
        获取导出数据（Fingers JSON 格式 - 数组）
        
        Returns:
            list: Fingers 格式的 JSON 数据（数组格式）
            [
                {"name": "...", "link": "...", "rule": [...], "tag": [...], 
                 "focus": false, "default_port": [...]},
                ...
            ]
        """
        fingerprints = self.model.objects.all()
        data = []
        for fp in fingerprints:
            item = {
                'name': fp.name,
                'link': fp.link,
                'rule': fp.rule,
                'tag': fp.tag,
            }
            # 只有当 focus 为 True 时才添加该字段（保持与原始格式一致）
            if fp.focus:
                item['focus'] = fp.focus
            # 只有当 default_port 非空时才添加该字段
            if fp.default_port:
                item['default_port'] = fp.default_port
            data.append(item)
        return data
