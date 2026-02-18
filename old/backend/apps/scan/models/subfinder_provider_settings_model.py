"""Subfinder Provider 配置模型（单例模式）

用于存储 subfinder 第三方数据源的 API Key 配置
"""

from django.db import models


class SubfinderProviderSettings(models.Model):
    """
    Subfinder Provider 配置（单例模式）
    存储第三方数据源的 API Key 配置，用于 subfinder 子域名发现
    
    支持的 Provider:
    - fofa: email + api_key (composite)
    - censys: api_id + api_secret (composite)
    - hunter, shodan, zoomeye, securitytrails, threatbook, quake: api_key (single)
    """
    
    providers = models.JSONField(
        default=dict,
        help_text='各 Provider 的 API Key 配置'
    )
    
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)
    
    class Meta:
        db_table = 'subfinder_provider_settings'
        verbose_name = 'Subfinder Provider 配置'
        verbose_name_plural = 'Subfinder Provider 配置'
    
    DEFAULT_PROVIDERS = {
        'fofa': {'enabled': False, 'email': '', 'api_key': ''},
        'hunter': {'enabled': False, 'api_key': ''},
        'shodan': {'enabled': False, 'api_key': ''},
        'censys': {'enabled': False, 'api_id': '', 'api_secret': ''},
        'zoomeye': {'enabled': False, 'api_key': ''},
        'securitytrails': {'enabled': False, 'api_key': ''},
        'threatbook': {'enabled': False, 'api_key': ''},
        'quake': {'enabled': False, 'api_key': ''},
    }
    
    def save(self, *args, **kwargs):
        self.pk = 1
        super().save(*args, **kwargs)
    
    @classmethod
    def get_instance(cls) -> 'SubfinderProviderSettings':
        """获取或创建单例实例"""
        obj, _ = cls.objects.get_or_create(
            pk=1,
            defaults={'providers': cls.DEFAULT_PROVIDERS.copy()}
        )
        return obj
    
    def get_provider_config(self, provider: str) -> dict:
        """获取指定 Provider 的配置"""
        return self.providers.get(provider, self.DEFAULT_PROVIDERS.get(provider, {}))
    
    def is_provider_enabled(self, provider: str) -> bool:
        """检查指定 Provider 是否启用"""
        config = self.get_provider_config(provider)
        return config.get('enabled', False)
