"""ARL 指纹 Serializer"""

from rest_framework import serializers

from apps.engine.models import ARLFingerprint


class ARLFingerprintSerializer(serializers.ModelSerializer):
    """ARL 指纹序列化器
    
    字段映射:
    - name: 指纹名称 (必填, 唯一)
    - rule: 匹配规则表达式 (必填)
    """
    
    class Meta:
        model = ARLFingerprint
        fields = ['id', 'name', 'rule', 'created_at']
        read_only_fields = ['id', 'created_at']
    
    def validate_name(self, value):
        """校验 name 字段"""
        if not value or not value.strip():
            raise serializers.ValidationError("name 字段不能为空")
        return value.strip()
    
    def validate_rule(self, value):
        """校验 rule 字段"""
        if not value or not value.strip():
            raise serializers.ValidationError("rule 字段不能为空")
        return value.strip()
