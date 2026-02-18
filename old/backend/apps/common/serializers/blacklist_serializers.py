"""黑名单规则序列化器"""
from rest_framework import serializers

from apps.common.models import BlacklistRule
from apps.common.utils import detect_rule_type


class BlacklistRuleSerializer(serializers.ModelSerializer):
    """黑名单规则序列化器"""
    
    class Meta:
        model = BlacklistRule
        fields = [
            'id',
            'pattern',
            'rule_type',
            'scope',
            'target',
            'description',
            'created_at',
        ]
        read_only_fields = ['id', 'rule_type', 'created_at']
    
    def validate_pattern(self, value):
        """验证规则模式"""
        if not value or not value.strip():
            raise serializers.ValidationError("规则模式不能为空")
        return value.strip()
    
    def create(self, validated_data):
        """创建规则时自动识别规则类型"""
        pattern = validated_data.get('pattern', '')
        validated_data['rule_type'] = detect_rule_type(pattern)
        return super().create(validated_data)
    
    def update(self, instance, validated_data):
        """更新规则时重新识别规则类型"""
        if 'pattern' in validated_data:
            pattern = validated_data['pattern']
            validated_data['rule_type'] = detect_rule_type(pattern)
        return super().update(instance, validated_data)


class GlobalBlacklistRuleSerializer(BlacklistRuleSerializer):
    """全局黑名单规则序列化器"""
    
    class Meta(BlacklistRuleSerializer.Meta):
        fields = ['id', 'pattern', 'rule_type', 'description', 'created_at']
        read_only_fields = ['id', 'rule_type', 'created_at']
    
    def create(self, validated_data):
        """创建全局规则"""
        validated_data['scope'] = BlacklistRule.Scope.GLOBAL
        validated_data['target'] = None
        return super().create(validated_data)


class TargetBlacklistRuleSerializer(BlacklistRuleSerializer):
    """Target 黑名单规则序列化器"""
    
    class Meta(BlacklistRuleSerializer.Meta):
        fields = ['id', 'pattern', 'rule_type', 'description', 'created_at']
        read_only_fields = ['id', 'rule_type', 'created_at']
    
    def create(self, validated_data):
        """创建 Target 规则（target_id 由 view 设置）"""
        validated_data['scope'] = BlacklistRule.Scope.TARGET
        return super().create(validated_data)
