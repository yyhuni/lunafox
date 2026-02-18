"""Fingers 指纹 Serializer"""

from rest_framework import serializers

from apps.engine.models import FingersFingerprint


class FingersFingerprintSerializer(serializers.ModelSerializer):
    """Fingers 指纹序列化器
    
    字段映射:
    - name: 指纹名称 (必填, 唯一)
    - link: 相关链接 (可选)
    - rule: 匹配规则数组 (必填)
    - tag: 标签数组 (可选)
    - focus: 是否重点关注 (可选, 默认 False)
    - default_port: 默认端口数组 (可选)
    """
    
    class Meta:
        model = FingersFingerprint
        fields = ['id', 'name', 'link', 'rule', 'tag', 'focus', 
                  'default_port', 'created_at']
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
    
    def validate_tag(self, value):
        """校验 tag 字段"""
        if not isinstance(value, list):
            raise serializers.ValidationError("tag 必须是数组")
        return value
    
    def validate_default_port(self, value):
        """校验 default_port 字段"""
        if not isinstance(value, list):
            raise serializers.ValidationError("default_port 必须是数组")
        return value
