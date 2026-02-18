"""Fingers 指纹管理 ViewSet"""

from apps.common.pagination import BasePagination
from apps.engine.models import FingersFingerprint
from apps.engine.serializers.fingerprints import FingersFingerprintSerializer
from apps.engine.services.fingerprints import FingersFingerprintService

from .base import BaseFingerprintViewSet


class FingersFingerprintViewSet(BaseFingerprintViewSet):
    """Fingers 指纹管理 ViewSet
    
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
    - name=="WordPress"  精确匹配
    - tag="cms"          按标签筛选
    - focus="true"       按重点关注筛选
    """
    
    queryset = FingersFingerprint.objects.all()
    serializer_class = FingersFingerprintSerializer
    pagination_class = BasePagination
    service_class = FingersFingerprintService
    
    # 排序配置
    ordering_fields = ['created_at', 'name']
    ordering = ['-created_at']
    
    # Fingers 过滤字段映射
    FILTER_FIELD_MAPPING = {
        'name': 'name',
        'link': 'link',
        'focus': 'focus',
    }
    
    # JSON 数组字段（使用 __contains 查询）
    JSON_ARRAY_FIELDS = ['tag', 'rule', 'default_port']
    
    def parse_import_data(self, json_data) -> list:
        """
        解析 Fingers JSON 格式的导入数据
        
        输入格式：[{...}, {...}] 数组格式
        返回：指纹列表
        """
        if isinstance(json_data, list):
            return json_data
        return []
    
    def get_export_filename(self) -> str:
        """导出文件名"""
        return 'fingers_http.json'
