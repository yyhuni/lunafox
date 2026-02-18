"""Wappalyzer 指纹管理 ViewSet"""

from apps.common.pagination import BasePagination
from apps.engine.models import WappalyzerFingerprint
from apps.engine.serializers.fingerprints import WappalyzerFingerprintSerializer
from apps.engine.services.fingerprints import WappalyzerFingerprintService

from .base import BaseFingerprintViewSet


class WappalyzerFingerprintViewSet(BaseFingerprintViewSet):
    """Wappalyzer 指纹管理 ViewSet
    
    继承自 BaseFingerprintViewSet，提供以下 API：
    
    标准 CRUD（ModelViewSet）：
    - GET    /                  列表查询（分页）
    - POST   /                  创建单条
    - GET    /{id}/             获取详情
    - PUT    /{id}/             更新
    - DELETE /{id}/             删除
    
    批量操作（继承自基类）：
    - POST   /batch_create/     批量创建（JSON body）
    - POST   /import_file/      文件导入（multipart/form-data）
    - POST   /bulk-delete/      批量删除
    - POST   /delete-all/       删除所有
    - GET    /export/           导出下载
    
    智能过滤语法（filter 参数）：
    - name="word"        模糊匹配 name 字段
    - name=="AppName"    精确匹配
    """
    
    queryset = WappalyzerFingerprint.objects.all()
    serializer_class = WappalyzerFingerprintSerializer
    pagination_class = BasePagination
    service_class = WappalyzerFingerprintService
    
    # 排序配置
    ordering_fields = ['created_at', 'name']
    ordering = ['-created_at']
    
    # Wappalyzer 过滤字段映射
    # 注意：implies 是 JSON 数组字段，使用 __contains 查询
    FILTER_FIELD_MAPPING = {
        'name': 'name',
        'description': 'description',
        'website': 'website',
        'cpe': 'cpe',
        'implies': 'implies',  # JSON 数组字段
    }
    
    # JSON 数组字段列表（使用 __contains 查询）
    JSON_ARRAY_FIELDS = ['implies']
    
    def parse_import_data(self, json_data: dict) -> list:
        """
        解析 Wappalyzer JSON 格式的导入数据
        
        Wappalyzer 格式是 apps 对象格式：{"apps": {"AppName": {...}, ...}}
        
        输入格式：{"apps": {"1C-Bitrix": {"cats": [...], ...}, ...}}
        返回：指纹列表（每个 app 转换为带 name 字段的 dict）
        """
        apps = json_data.get('apps', {})
        fingerprints = []
        for name, data in apps.items():
            item = {'name': name, **data}
            fingerprints.append(item)
        return fingerprints
    
    def get_export_filename(self) -> str:
        """导出文件名"""
        return 'wappalyzer.json'
