"""
集中式权限管理

实现三类端点的认证逻辑：
1. 公开端点（无需认证）：登录、登出、获取当前用户状态
2. Worker 端点（API Key 认证）：注册、配置、心跳、回调、资源同步
3. 业务端点（Session 认证）：其他所有 API
"""

import re
import logging
from django.conf import settings
from rest_framework.permissions import BasePermission

logger = logging.getLogger(__name__)

# 公开端点白名单（无需任何认证）
PUBLIC_ENDPOINTS = [
    r'^/api/auth/login/$',
    r'^/api/auth/logout/$',
    r'^/api/auth/me/$',
]

# Worker API 端点（需要 API Key 认证）
# 包括：注册、配置、心跳、回调、资源同步（字典下载）
WORKER_ENDPOINTS = [
    r'^/api/workers/register/$',
    r'^/api/workers/config/$',
    r'^/api/workers/\d+/heartbeat/$',
    r'^/api/callbacks/',
    # 资源同步端点（Worker 需要下载字典文件）
    r'^/api/wordlists/download/$',
    # 注意：指纹导出 API 使用 Session 认证（前端用户导出用）
    # Worker 通过数据库直接获取指纹数据，不需要 HTTP API
]


class IsAuthenticatedOrPublic(BasePermission):
    """
    自定义权限类：
    - 白名单内的端点公开访问
    - Worker 端点需要 API Key 认证
    - 其他端点需要 Session 认证
    """
    
    def has_permission(self, request, view):
        path = request.path
        
        # 检查是否在公开白名单内
        for pattern in PUBLIC_ENDPOINTS:
            if re.match(pattern, path):
                return True
        
        # 检查是否是 Worker 端点
        for pattern in WORKER_ENDPOINTS:
            if re.match(pattern, path):
                return self._check_worker_api_key(request)
        
        # 其他路径需要 Session 认证
        return request.user and request.user.is_authenticated
    
    def _check_worker_api_key(self, request):
        """验证 Worker API Key"""
        api_key = request.headers.get('X-Worker-API-Key')
        expected_key = getattr(settings, 'WORKER_API_KEY', None)
        
        if not expected_key:
            # 未配置 API Key 时，拒绝所有 Worker 请求
            logger.warning("WORKER_API_KEY 未配置，拒绝 Worker 请求")
            return False
        
        if not api_key:
            logger.warning(f"Worker 请求缺少 X-Worker-API-Key Header: {request.path}")
            return False
        
        if api_key != expected_key:
            logger.warning(f"Worker API Key 无效: {request.path}")
            return False
        
        return True
