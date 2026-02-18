"""Wappalyzer 指纹管理 Service

实现 Wappalyzer 格式指纹的校验、转换和导出逻辑
"""

from apps.engine.models import WappalyzerFingerprint
from .base import BaseFingerprintService


class WappalyzerFingerprintService(BaseFingerprintService):
    """Wappalyzer 指纹管理服务（继承基类，实现 Wappalyzer 特定逻辑）"""
    
    model = WappalyzerFingerprint
    
    def validate_fingerprint(self, item: dict) -> bool:
        """
        校验单条 Wappalyzer 指纹
        
        校验规则：
        - name 字段必须存在且非空（从 apps 对象的 key 传入）
        
        Args:
            item: 单条指纹数据
            
        Returns:
            bool: 是否有效
        """
        name = item.get('name', '')
        return bool(name and str(name).strip())
    
    def to_model_data(self, item: dict) -> dict:
        """
        转换 Wappalyzer JSON 格式为 Model 字段
        
        字段映射：
        - scriptSrc (JSON) → script_src (Model)
        
        Args:
            item: 原始 Wappalyzer JSON 数据
            
        Returns:
            dict: Model 字段数据
        """
        return {
            'name': str(item.get('name', '')).strip(),
            'cats': item.get('cats', []),
            'cookies': item.get('cookies', {}),
            'headers': item.get('headers', {}),
            'script_src': item.get('scriptSrc', []),  # JSON: scriptSrc -> Model: script_src
            'js': item.get('js', []),
            'implies': item.get('implies', []),
            'meta': item.get('meta', {}),
            'html': item.get('html', []),
            'description': item.get('description', ''),
            'website': item.get('website', ''),
            'cpe': item.get('cpe', ''),
        }
    
    def get_export_data(self) -> dict:
        """
        获取导出数据（Wappalyzer JSON 格式）
        
        Returns:
            dict: Wappalyzer 格式的 JSON 数据
            {
                "apps": {
                    "AppName": {"cats": [...], "cookies": {...}, ...},
                    ...
                }
            }
        """
        fingerprints = self.model.objects.all()
        apps = {}
        for fp in fingerprints:
            app_data = {}
            if fp.cats:
                app_data['cats'] = fp.cats
            if fp.cookies:
                app_data['cookies'] = fp.cookies
            if fp.headers:
                app_data['headers'] = fp.headers
            if fp.script_src:
                app_data['scriptSrc'] = fp.script_src  # Model: script_src -> JSON: scriptSrc
            if fp.js:
                app_data['js'] = fp.js
            if fp.implies:
                app_data['implies'] = fp.implies
            if fp.meta:
                app_data['meta'] = fp.meta
            if fp.html:
                app_data['html'] = fp.html
            if fp.description:
                app_data['description'] = fp.description
            if fp.website:
                app_data['website'] = fp.website
            if fp.cpe:
                app_data['cpe'] = fp.cpe
            apps[fp.name] = app_data
        return {'apps': apps}
