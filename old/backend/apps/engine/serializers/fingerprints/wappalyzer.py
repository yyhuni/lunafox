"""Wappalyzer 指纹 Serializer"""

from rest_framework import serializers

from apps.engine.models import WappalyzerFingerprint


class WappalyzerFingerprintSerializer(serializers.ModelSerializer):
    """Wappalyzer 指纹序列化器"""
    
    class Meta:
        model = WappalyzerFingerprint
        fields = [
            'id', 'name', 'cats', 'cookies', 'headers', 'script_src',
            'js', 'implies', 'meta', 'html', 'description', 'website',
            'cpe', 'created_at'
        ]
        read_only_fields = ['id', 'created_at']
    
    def validate_name(self, value):
        """校验 name 字段"""
        if not value or not value.strip():
            raise serializers.ValidationError("name 字段不能为空")
        return value.strip()
