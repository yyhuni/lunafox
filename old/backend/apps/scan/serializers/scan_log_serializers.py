"""扫描日志序列化器"""

from rest_framework import serializers

from ..models import ScanLog


class ScanLogSerializer(serializers.ModelSerializer):
    """扫描日志序列化器"""
    
    class Meta:
        model = ScanLog
        fields = ['id', 'level', 'content', 'created_at']
