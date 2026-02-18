"""ARL 指纹管理 ViewSet"""

import yaml
from django.http import HttpResponse
from rest_framework.decorators import action
from rest_framework.exceptions import ValidationError

from apps.common.pagination import BasePagination
from apps.common.response_helpers import success_response
from apps.engine.models import ARLFingerprint
from apps.engine.serializers.fingerprints import ARLFingerprintSerializer
from apps.engine.services.fingerprints import ARLFingerprintService

from .base import BaseFingerprintViewSet


class ARLFingerprintViewSet(BaseFingerprintViewSet):
    """ARL 指纹管理 ViewSet
    
    继承自 BaseFingerprintViewSet，提供以下 API：
    
    标准 CRUD（ModelViewSet）：
    - GET    /                  列表查询（分页）
    - POST   /                  创建单条
    - GET    /{id}/             获取详情
    - PUT    /{id}/             更新
    - DELETE /{id}/             删除
    
    批量操作（继承自基类）：
    - POST   /batch_create/     批量创建（JSON body）
    - POST   /import_file/      文件导入（multipart/form-data，支持 YAML）
    - POST   /bulk-delete/      批量删除
    - POST   /delete-all/       删除所有
    - GET    /export/           导出下载（YAML 格式）
    
    智能过滤语法（filter 参数）：
    - name="word"        模糊匹配 name 字段
    - name=="WordPress"  精确匹配
    - rule="body="       按规则内容筛选
    """
    
    queryset = ARLFingerprint.objects.all()
    serializer_class = ARLFingerprintSerializer
    pagination_class = BasePagination
    service_class = ARLFingerprintService
    
    # 排序配置
    ordering_fields = ['created_at', 'name']
    ordering = ['-created_at']
    
    # ARL 过滤字段映射
    FILTER_FIELD_MAPPING = {
        'name': 'name',
        'rule': 'rule',
    }
    
    def parse_import_data(self, json_data) -> list:
        """
        解析 ARL 格式的导入数据（JSON 格式）
        
        输入格式：[{...}, {...}] 数组格式
        返回：指纹列表
        """
        if isinstance(json_data, list):
            return json_data
        return []
    
    def get_export_filename(self) -> str:
        """导出文件名"""
        return 'ARL.yaml'
    
    @action(detail=False, methods=['post'])
    def import_file(self, request):
        """
        文件导入（支持 YAML 和 JSON 格式）
        POST /api/engine/fingerprints/arl/import_file/
        
        请求格式：multipart/form-data
        - file: YAML 或 JSON 文件
        
        返回：同 batch_create
        """
        file = request.FILES.get('file')
        if not file:
            raise ValidationError('缺少文件')
        
        filename = file.name.lower()
        content = file.read().decode('utf-8')
        
        try:
            if filename.endswith('.yaml') or filename.endswith('.yml'):
                # YAML 格式
                fingerprints = yaml.safe_load(content)
            else:
                # JSON 格式
                import json
                fingerprints = json.loads(content)
        except (yaml.YAMLError, json.JSONDecodeError) as e:
            raise ValidationError(f'无效的文件格式: {e}')
        
        if not isinstance(fingerprints, list):
            raise ValidationError('文件内容必须是数组格式')
        
        if not fingerprints:
            raise ValidationError('文件中没有有效的指纹数据')
        
        result = self.get_service().batch_create_fingerprints(fingerprints)
        return success_response(data=result)
    
    @action(detail=False, methods=['get'])
    def export(self, request):
        """
        导出指纹（YAML 格式）
        GET /api/engine/fingerprints/arl/export/
        
        返回：YAML 文件下载
        """
        data = self.get_service().get_export_data()
        content = yaml.dump(data, allow_unicode=True, default_flow_style=False, sort_keys=False)
        response = HttpResponse(content, content_type='application/x-yaml')
        response['Content-Disposition'] = f'attachment; filename="{self.get_export_filename()}"'
        return response
