"""序列化器通用 Mixin 和工具类"""

from rest_framework import serializers
import yaml


class DuplicateKeyLoader(yaml.SafeLoader):
    """自定义 YAML Loader，检测重复 key"""
    pass


def _check_duplicate_keys(loader, node, deep=False):
    """检测 YAML mapping 中的重复 key"""
    mapping = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in mapping:
            raise yaml.constructor.ConstructorError(
                "while constructing a mapping", node.start_mark,
                f"发现重复的配置项 '{key}'，后面的配置会覆盖前面的配置，请删除重复项", key_node.start_mark
            )
        mapping[key] = loader.construct_object(value_node, deep=deep)
    return mapping


DuplicateKeyLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG,
    _check_duplicate_keys
)


class ScanConfigValidationMixin:
    """扫描配置验证 Mixin"""
    
    def validate_configuration(self, value):
        """验证 YAML 配置格式"""
        if not value or not value.strip():
            raise serializers.ValidationError("configuration 不能为空")
        
        try:
            yaml.load(value, Loader=DuplicateKeyLoader)
        except yaml.YAMLError as e:
            raise serializers.ValidationError(f"无效的 YAML 格式: {str(e)}")
        
        return value
    
    def validate_engine_ids(self, value):
        """验证引擎 ID 列表"""
        if not value:
            raise serializers.ValidationError("engine_ids 不能为空，请至少选择一个扫描引擎")
        return value
    
    def validate_engine_names(self, value):
        """验证引擎名称列表"""
        if not value:
            raise serializers.ValidationError("engine_names 不能为空")
        return value
