"""Subfinder Provider 配置文件生成服务

负责生成 subfinder 的 provider-config.yaml 配置文件
"""

import logging
import os
from pathlib import Path
from typing import Optional

import yaml

from ..models import SubfinderProviderSettings

logger = logging.getLogger(__name__)


class SubfinderProviderConfigService:
    """Subfinder Provider 配置文件生成服务"""
    
    # Provider 格式定义
    PROVIDER_FORMATS = {
        'fofa': {'type': 'composite', 'format': '{email}:{api_key}'},
        'censys': {'type': 'composite', 'format': '{api_id}:{api_secret}'},
        'hunter': {'type': 'single', 'field': 'api_key'},
        'shodan': {'type': 'single', 'field': 'api_key'},
        'zoomeye': {'type': 'single', 'field': 'api_key'},
        'securitytrails': {'type': 'single', 'field': 'api_key'},
        'threatbook': {'type': 'single', 'field': 'api_key'},
        'quake': {'type': 'single', 'field': 'api_key'},
    }
    
    def generate(self, output_dir: str) -> Optional[str]:
        """
        生成 provider-config.yaml 文件
        
        Args:
            output_dir: 输出目录路径
            
        Returns:
            生成的配置文件路径，如果没有启用的 provider 则返回 None
        """
        settings = SubfinderProviderSettings.get_instance()
        
        config = {}
        has_enabled = False
        
        for provider, format_info in self.PROVIDER_FORMATS.items():
            provider_config = settings.providers.get(provider, {})
            
            if not provider_config.get('enabled'):
                config[provider] = []
                continue
            
            value = self._build_provider_value(provider, provider_config)
            if value:
                config[provider] = [value]  # 单个 key 放入数组
                has_enabled = True
                logger.debug(f"Provider {provider} 已启用")
            else:
                config[provider] = []
        
        # 检查是否有任何启用的 provider
        if not has_enabled:
            logger.info("没有启用的 Provider，跳过配置文件生成")
            return None
        
        # 确保输出目录存在
        output_path = Path(output_dir) / 'provider-config.yaml'
        output_path.parent.mkdir(parents=True, exist_ok=True)
        
        # 写入 YAML 文件（使用默认列表格式，和 subfinder 一致）
        with open(output_path, 'w', encoding='utf-8') as f:
            yaml.dump(config, f, default_flow_style=False, allow_unicode=True)
        
        # 设置文件权限为 600（仅所有者可读写）
        os.chmod(output_path, 0o600)
        
        logger.info(f"Provider 配置文件已生成: {output_path}")
        return str(output_path)
    
    def _build_provider_value(self, provider: str, config: dict) -> Optional[str]:
        """根据 provider 格式规则构建配置值
        
        Args:
            provider: provider 名称
            config: provider 配置字典
            
        Returns:
            构建的配置值字符串，如果配置不完整则返回 None
        """
        format_info = self.PROVIDER_FORMATS.get(provider)
        if not format_info:
            return None
        
        if format_info['type'] == 'composite':
            # 复合格式：需要多个字段
            format_str = format_info['format']
            try:
                # 提取格式字符串中的字段名
                # 例如 '{email}:{api_key}' -> ['email', 'api_key']
                import re
                fields = re.findall(r'\{(\w+)\}', format_str)
                
                # 检查所有字段是否都有值
                values = {}
                for field in fields:
                    value = config.get(field, '').strip()
                    if not value:
                        logger.debug(f"Provider {provider} 缺少字段 {field}")
                        return None
                    values[field] = value
                
                return format_str.format(**values)
            except (KeyError, ValueError) as e:
                logger.warning(f"构建 {provider} 配置值失败: {e}")
                return None
        else:
            # 单字段格式
            field = format_info['field']
            value = config.get(field, '').strip()
            if not value:
                logger.debug(f"Provider {provider} 缺少字段 {field}")
                return None
            return value
    
    def cleanup(self, config_path: str) -> None:
        """清理配置文件
        
        Args:
            config_path: 配置文件路径
        """
        try:
            if config_path and Path(config_path).exists():
                Path(config_path).unlink()
                logger.debug(f"已清理配置文件: {config_path}")
        except Exception as e:
            logger.warning(f"清理配置文件失败: {config_path} - {e}")
