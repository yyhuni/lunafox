"""Goby 指纹 Serializer"""

from rest_framework import serializers

from apps.engine.models import GobyFingerprint


class GobyFingerprintSerializer(serializers.ModelSerializer):
    """Goby 指纹序列化器"""
    
    class Meta:
        model = GobyFingerprint
        fields = ['id', 'name', 'logic', 'rule', 'created_at']
        read_only_fields = ['id', 'created_at']
    
    def validate_name(self, value):
        """校验 name 字段"""
        if not value or not value.strip():
            raise serializers.ValidationError("name 字段不能为空")
        return value.strip()
    
    def validate_rule(self, value):
        """校验 rule 字段"""
        if not isinstance(value, list):
            raise serializers.ValidationError("rule 必须是数组")
        return value
