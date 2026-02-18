"""FingerPrintHub 指纹 Serializer"""

from rest_framework import serializers

from apps.engine.models import FingerPrintHubFingerprint


class FingerPrintHubFingerprintSerializer(serializers.ModelSerializer):
    """FingerPrintHub 指纹序列化器
    
    字段映射:
    - fp_id: 指纹ID (必填, 唯一)
    - name: 指纹名称 (必填)
    - author: 作者 (可选)
    - tags: 标签字符串 (可选)
    - severity: 严重程度 (可选, 默认 'info')
    - metadata: 元数据 JSON (可选)
    - http: HTTP 匹配规则数组 (必填)
    - source_file: 来源文件 (可选)
    """
    
    class Meta:
        model = FingerPrintHubFingerprint
        fields = ['id', 'fp_id', 'name', 'author', 'tags', 'severity',
                  'metadata', 'http', 'source_file', 'created_at']
        read_only_fields = ['id', 'created_at']
    
    def validate_fp_id(self, value):
        """校验 fp_id 字段"""
        if not value or not value.strip():
            raise serializers.ValidationError("fp_id 字段不能为空")
        return value.strip()
    
    def validate_name(self, value):
        """校验 name 字段"""
        if not value or not value.strip():
            raise serializers.ValidationError("name 字段不能为空")
        return value.strip()
    
    def validate_http(self, value):
        """校验 http 字段"""
        if not isinstance(value, list):
            raise serializers.ValidationError("http 必须是数组")
        return value
    
    def validate_metadata(self, value):
        """校验 metadata 字段"""
        if not isinstance(value, dict):
            raise serializers.ValidationError("metadata 必须是对象")
        return value
