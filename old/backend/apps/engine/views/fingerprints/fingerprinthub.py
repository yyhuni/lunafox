"""FingerPrintHub 指纹管理 ViewSet"""

from apps.common.pagination import BasePagination
from apps.engine.models import FingerPrintHubFingerprint
from apps.engine.serializers.fingerprints import FingerPrintHubFingerprintSerializer
from apps.engine.services.fingerprints import FingerPrintHubFingerprintService

from .base import BaseFingerprintViewSet


class FingerPrintHubFingerprintViewSet(BaseFingerprintViewSet):
    """FingerPrintHub 指纹管理 ViewSet
    
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
    - fp_id=="xxx"       精确匹配指纹ID
    - author="xxx"       按作者筛选
    - severity="info"    按严重程度筛选
    - tags="cms"         按标签筛选
    """
    
    queryset = FingerPrintHubFingerprint.objects.all()
    serializer_class = FingerPrintHubFingerprintSerializer
    pagination_class = BasePagination
    service_class = FingerPrintHubFingerprintService
    
    # 排序配置
    ordering_fields = ['created_at', 'name', 'severity']
    ordering = ['-created_at']
    
    # FingerPrintHub 过滤字段映射
    FILTER_FIELD_MAPPING = {
        'fp_id': 'fp_id',
        'name': 'name',
        'author': 'author',
        'tags': 'tags',
        'severity': 'severity',
        'source_file': 'source_file',
    }
    
    # JSON 数组字段（使用 __contains 查询）
    JSON_ARRAY_FIELDS = ['http']
    
    def parse_import_data(self, json_data) -> list:
        """
        解析 FingerPrintHub JSON 格式的导入数据
        
        输入格式：[{...}, {...}] 数组格式
        返回：指纹列表
        """
        if isinstance(json_data, list):
            return json_data
        return []
    
    def get_export_filename(self) -> str:
        """导出文件名"""
        return 'fingerprinthub_web.json'
