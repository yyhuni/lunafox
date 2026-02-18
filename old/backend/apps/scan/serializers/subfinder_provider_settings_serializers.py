"""Subfinder Provider 配置序列化器"""

from rest_framework import serializers


class SubfinderProviderSettingsSerializer(serializers.Serializer):
    """Subfinder Provider 配置序列化器
    
    支持的 Provider:
    - fofa: email + api_key (composite)
    - censys: api_id + api_secret (composite)
    - hunter, shodan, zoomeye, securitytrails, threatbook, quake: api_key (single)
    
    注意：djangorestframework-camel-case 会自动处理 camelCase <-> snake_case 转换
    所以这里统一使用 snake_case
    """
    
    VALID_PROVIDERS = {
        'fofa', 'hunter', 'shodan', 'censys', 
        'zoomeye', 'securitytrails', 'threatbook', 'quake'
    }
    
    def to_internal_value(self, data):
        """验证并转换输入数据"""
        if not isinstance(data, dict):
            raise serializers.ValidationError('Expected a dictionary')
        
        result = {}
        for provider, config in data.items():
            if provider not in self.VALID_PROVIDERS:
                continue
            
            if not isinstance(config, dict):
                continue
            
            db_config = {'enabled': bool(config.get('enabled', False))}
            
            if provider == 'fofa':
                db_config['email'] = str(config.get('email', ''))
                db_config['api_key'] = str(config.get('api_key', ''))
            elif provider == 'censys':
                db_config['api_id'] = str(config.get('api_id', ''))
                db_config['api_secret'] = str(config.get('api_secret', ''))
            else:
                db_config['api_key'] = str(config.get('api_key', ''))
            
            result[provider] = db_config
        
        return result
    
    def to_representation(self, instance):
        """输出数据（数据库格式，camel-case 中间件会自动转换）"""
        if isinstance(instance, dict):
            return instance
        return instance.providers if hasattr(instance, 'providers') else {}
