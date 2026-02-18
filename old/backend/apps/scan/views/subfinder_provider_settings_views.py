"""Subfinder Provider 配置视图"""

import logging
from rest_framework import status
from rest_framework.views import APIView
from rest_framework.response import Response

from ..models import SubfinderProviderSettings
from ..serializers import SubfinderProviderSettingsSerializer

logger = logging.getLogger(__name__)


class SubfinderProviderSettingsView(APIView):
    """Subfinder Provider 配置视图
    
    GET /api/settings/api-keys/ - 获取配置
    PUT /api/settings/api-keys/ - 更新配置
    """
    
    def get(self, request):
        """获取 Subfinder Provider 配置"""
        settings = SubfinderProviderSettings.get_instance()
        serializer = SubfinderProviderSettingsSerializer(settings.providers)
        return Response(serializer.data)
    
    def put(self, request):
        """更新 Subfinder Provider 配置"""
        serializer = SubfinderProviderSettingsSerializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        
        settings = SubfinderProviderSettings.get_instance()
        settings.providers.update(serializer.validated_data)
        settings.save()
        
        logger.info("Subfinder Provider 配置已更新")
        
        return Response(SubfinderProviderSettingsSerializer(settings.providers).data)
