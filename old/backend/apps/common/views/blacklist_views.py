"""全局黑名单 API 视图"""
import logging

from rest_framework import status
from rest_framework.views import APIView
from rest_framework.permissions import IsAuthenticated

from apps.common.response_helpers import success_response, error_response
from apps.common.services import BlacklistService

logger = logging.getLogger(__name__)


class GlobalBlacklistView(APIView):
    """
    全局黑名单规则 API
    
    Endpoints:
    - GET /api/blacklist/rules/ - 获取全局黑名单列表
    - PUT /api/blacklist/rules/ - 全量替换规则（文本框保存场景）
    
    设计说明：
    - 使用 PUT 全量替换模式，适合"文本框每行一个规则"的前端场景
    - 用户编辑文本框 -> 点击保存 -> 后端全量替换
    
    架构：MVS 模式
    - View: 参数验证、响应格式化
    - Service: 业务逻辑（BlacklistService）
    - Model: 数据持久化（BlacklistRule）
    """
    
    permission_classes = [IsAuthenticated]
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.blacklist_service = BlacklistService()
    
    def get(self, request):
        """
        获取全局黑名单规则列表
        
        返回格式：
        {
            "patterns": ["*.gov", "*.edu", "10.0.0.0/8"]
        }
        """
        rules = self.blacklist_service.get_global_rules()
        patterns = list(rules.values_list('pattern', flat=True))
        return success_response(data={'patterns': patterns})
    
    def put(self, request):
        """
        全量替换全局黑名单规则
        
        请求格式：
        {
            "patterns": ["*.gov", "*.edu", "10.0.0.0/8"]
        }
        
        或者空数组清空所有规则：
        {
            "patterns": []
        }
        """
        patterns = request.data.get('patterns', [])
        
        # 兼容字符串输入（换行分隔）
        if isinstance(patterns, str):
            patterns = [p for p in patterns.split('\n') if p.strip()]
        
        if not isinstance(patterns, list):
            return error_response(
                code='VALIDATION_ERROR',
                message='patterns 必须是数组'
            )
        
        # 调用 Service 层全量替换
        result = self.blacklist_service.replace_global_rules(patterns)
        
        return success_response(data=result)
