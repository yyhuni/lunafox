"""定时扫描序列化器"""

from rest_framework import serializers

from ..models import ScheduledScan
from .mixins import ScanConfigValidationMixin


class ScheduledScanSerializer(serializers.ModelSerializer):
    """定时扫描任务序列化器（用于列表和详情）"""
    
    organization_id = serializers.IntegerField(source='organization.id', read_only=True, allow_null=True)
    organization_name = serializers.CharField(source='organization.name', read_only=True, allow_null=True)
    target_id = serializers.IntegerField(source='target.id', read_only=True, allow_null=True)
    target_name = serializers.CharField(source='target.name', read_only=True, allow_null=True)
    scan_mode = serializers.SerializerMethodField()
    
    class Meta:
        model = ScheduledScan
        fields = [
            'id', 'name',
            'engine_ids', 'engine_names',
            'organization_id', 'organization_name',
            'target_id', 'target_name',
            'scan_mode',
            'cron_expression',
            'is_enabled',
            'run_count', 'last_run_time', 'next_run_time',
            'created_at', 'updated_at'
        ]
        read_only_fields = [
            'id', 'run_count',
            'last_run_time', 'next_run_time',
            'created_at', 'updated_at'
        ]
    
    def get_scan_mode(self, obj):
        return 'organization' if obj.organization_id else 'target'


class CreateScheduledScanSerializer(ScanConfigValidationMixin, serializers.Serializer):
    """创建定时扫描任务序列化器"""
    
    name = serializers.CharField(max_length=200, help_text='任务名称')
    configuration = serializers.CharField(required=True, help_text='YAML 格式的扫描配置')
    engine_ids = serializers.ListField(child=serializers.IntegerField(), required=True)
    engine_names = serializers.ListField(child=serializers.CharField(), required=True)
    organization_id = serializers.IntegerField(required=False, allow_null=True)
    target_id = serializers.IntegerField(required=False, allow_null=True)
    cron_expression = serializers.CharField(max_length=100, default='0 2 * * *')
    is_enabled = serializers.BooleanField(default=True)
    
    def validate(self, data):
        organization_id = data.get('organization_id')
        target_id = data.get('target_id')
        
        if not organization_id and not target_id:
            raise serializers.ValidationError('必须提供 organization_id 或 target_id 其中之一')
        if organization_id and target_id:
            raise serializers.ValidationError('organization_id 和 target_id 只能提供其中之一')
        
        return data


class UpdateScheduledScanSerializer(serializers.Serializer):
    """更新定时扫描任务序列化器"""
    
    name = serializers.CharField(max_length=200, required=False)
    engine_ids = serializers.ListField(child=serializers.IntegerField(), required=False)
    organization_id = serializers.IntegerField(required=False, allow_null=True)
    target_id = serializers.IntegerField(required=False, allow_null=True)
    cron_expression = serializers.CharField(max_length=100, required=False)
    is_enabled = serializers.BooleanField(required=False)
    
    def validate_engine_ids(self, value):
        if value is not None and not value:
            raise serializers.ValidationError("engine_ids 不能为空")
        return value


class ToggleScheduledScanSerializer(serializers.Serializer):
    """切换定时扫描启用状态序列化器"""
    
    is_enabled = serializers.BooleanField(help_text='是否启用')
