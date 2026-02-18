"""FingerPrintHub 指纹管理 Service

实现 FingerPrintHub 格式指纹的校验、转换和导出逻辑
"""

from apps.engine.models import FingerPrintHubFingerprint
from .base import BaseFingerprintService


class FingerPrintHubFingerprintService(BaseFingerprintService):
    """FingerPrintHub 指纹管理服务（继承基类，实现 FingerPrintHub 特定逻辑）"""
    
    model = FingerPrintHubFingerprint
    
    def validate_fingerprint(self, item: dict) -> bool:
        """
        校验单条 FingerPrintHub 指纹
        
        校验规则：
        - id 字段必须存在且非空
        - info 字段必须存在且包含 name
        - http 字段必须是数组
        
        Args:
            item: 单条指纹数据
            
        Returns:
            bool: 是否有效
        """
        fp_id = item.get('id', '')
        info = item.get('info', {})
        http = item.get('http')
        
        if not fp_id or not str(fp_id).strip():
            return False
        if not isinstance(info, dict) or not info.get('name'):
            return False
        if not isinstance(http, list):
            return False
        
        return True
    
    def to_model_data(self, item: dict) -> dict:
        """
        转换 FingerPrintHub JSON 格式为 Model 字段
        
        字段映射（嵌套结构转扁平）：
        - id (JSON) → fp_id (Model)
        - info.name (JSON) → name (Model)
        - info.author (JSON) → author (Model)
        - info.tags (JSON) → tags (Model)
        - info.severity (JSON) → severity (Model)
        - info.metadata (JSON) → metadata (Model)
        - http (JSON) → http (Model)
        - _source_file (JSON) → source_file (Model)
        
        Args:
            item: 原始 FingerPrintHub JSON 数据
            
        Returns:
            dict: Model 字段数据
        """
        info = item.get('info', {})
        return {
            'fp_id': str(item.get('id', '')).strip(),
            'name': str(info.get('name', '')).strip(),
            'author': info.get('author', ''),
            'tags': info.get('tags', ''),
            'severity': info.get('severity', 'info'),
            'metadata': info.get('metadata', {}),
            'http': item.get('http', []),
            'source_file': item.get('_source_file', ''),
        }
    
    def get_export_data(self) -> list:
        """
        获取导出数据（FingerPrintHub JSON 格式 - 数组）
        
        Returns:
            list: FingerPrintHub 格式的 JSON 数据（数组格式）
            [
                {
                    "id": "...",
                    "info": {"name": "...", "author": "...", "tags": "...", 
                             "severity": "...", "metadata": {...}},
                    "http": [...],
                    "_source_file": "..."
                },
                ...
            ]
        """
        fingerprints = self.model.objects.all()
        data = []
        for fp in fingerprints:
            item = {
                'id': fp.fp_id,
                'info': {
                    'name': fp.name,
                    'author': fp.author,
                    'tags': fp.tags,
                    'severity': fp.severity,
                    'metadata': fp.metadata,
                },
                'http': fp.http,
            }
            # 只有当 source_file 非空时才添加该字段
            if fp.source_file:
                item['_source_file'] = fp.source_file
            data.append(item)
        return data
