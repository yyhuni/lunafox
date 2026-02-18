"""Goby 指纹管理 ViewSet"""

from apps.common.pagination import BasePagination
from apps.engine.models import GobyFingerprint
from apps.engine.serializers.fingerprints import GobyFingerprintSerializer
from apps.engine.services.fingerprints import GobyFingerprintService

from .base import BaseFingerprintViewSet


class GobyFingerprintViewSet(BaseFingerprintViewSet):
    """Goby 指纹管理 ViewSet
    
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
    - name=="ProductName" 精确匹配
    """
    
    queryset = GobyFingerprint.objects.all()
    serializer_class = GobyFingerprintSerializer
    pagination_class = BasePagination
    service_class = GobyFingerprintService
    
    # 排序配置
    ordering_fields = ['created_at', 'name']
    ordering = ['-created_at']
    
    # Goby 过滤字段映射
    FILTER_FIELD_MAPPING = {
        'name': 'name',
        'logic': 'logic',
    }
    
    def parse_import_data(self, json_data) -> list:
        """
        解析 Goby JSON 格式的导入数据
        
        Goby 格式是数组格式：[{...}, {...}, ...]
        
        输入格式：[{"name": "...", "logic": "...", "rule": [...]}, ...]
        返回：指纹列表
        """
        if isinstance(json_data, list):
            return json_data
        return []
    
    def get_export_filename(self) -> str:
        """导出文件名"""
        return 'goby.json'
