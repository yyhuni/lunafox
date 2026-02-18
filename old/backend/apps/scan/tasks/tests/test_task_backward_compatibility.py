"""
Task 向后兼容性测试

Property 8: Task Backward Compatibility
*For any* 任务调用，当仅提供 target_id 参数时，任务应该创建 DatabaseTargetProvider 
并使用它进行数据访问，行为与改造前一致。

**Validates: Requirements 6.1, 6.2, 6.4, 6.5**
"""

import tempfile
import pytest
from pathlib import Path
from unittest.mock import patch, MagicMock
from hypothesis import given, strategies as st, settings

from apps.scan.tasks.port_scan.export_hosts_task import export_hosts_task
from apps.scan.tasks.site_scan.export_site_urls_task import export_site_urls_task
from apps.scan.providers import ListTargetProvider


# 生成有效域名的策略
def valid_domain_strategy():
    """生成有效的域名"""
    label = st.text(
        alphabet=st.characters(whitelist_categories=('L',), min_codepoint=97, max_codepoint=122),
        min_size=2,
        max_size=10
    )
    return st.builds(
        lambda a, b, c: f"{a}.{b}.{c}",
        label, label, st.sampled_from(['com', 'net', 'org', 'io'])
    )


class TestExportHostsTaskBackwardCompatibility:
    """export_hosts_task 向后兼容性测试"""
    
    @given(
        target_id=st.integers(min_value=1, max_value=1000),
        hosts=st.lists(valid_domain_strategy(), min_size=1, max_size=10)
    )
    @settings(max_examples=50, deadline=None)
    def test_property_8_legacy_mode_creates_database_provider(self, target_id, hosts):
        """
        Property 8: Task Backward Compatibility (export_hosts_task)
        
        Feature: scan-target-provider, Property 8: Task Backward Compatibility
        **Validates: Requirements 6.1, 6.2, 6.4, 6.5**
        
        For any target_id, when calling export_hosts_task with only target_id,
        it should create a DatabaseTargetProvider and use it for data access.
        """
        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.txt') as f:
            output_file = f.name
        
        try:
            # Mock Target 和 SubdomainService
            mock_target = MagicMock()
            mock_target.type = 'domain'
            mock_target.name = hosts[0]
            
            with patch('apps.scan.tasks.port_scan.export_hosts_task.DatabaseTargetProvider') as mock_provider_class, \
                 patch('apps.targets.services.TargetService') as mock_target_service:
                
                # 创建 mock provider 实例
                mock_provider = MagicMock()
                mock_provider.iter_hosts.return_value = iter(hosts)
                mock_provider.get_blacklist_filter.return_value = None
                mock_provider_class.return_value = mock_provider
                
                # Mock TargetService
                mock_target_service.return_value.get_target.return_value = mock_target
                
                # 调用任务（传统模式：只传 target_id）
                result = export_hosts_task(
                    output_file=output_file,
                    target_id=target_id
                )
                
                # 验证：应该创建了 DatabaseTargetProvider
                mock_provider_class.assert_called_once_with(target_id=target_id)
                
                # 验证：返回值包含必需字段
                assert result['success'] is True
                assert result['output_file'] == output_file
                assert result['total_count'] == len(hosts)
                assert 'target_type' in result  # 传统模式应该返回 target_type
                
                # 验证：文件内容正确
                with open(output_file, 'r') as f:
                    lines = [line.strip() for line in f.readlines()]
                    assert lines == hosts
        
        finally:
            Path(output_file).unlink(missing_ok=True)
    
    def test_legacy_mode_with_provider_parameter(self):
        """测试当同时提供 target_id 和 provider 时，provider 优先"""
        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.txt') as f:
            output_file = f.name
        
        try:
            hosts = ['example.com', 'test.com']
            provider = ListTargetProvider(targets=hosts)
            
            # 调用任务（同时提供 target_id 和 provider）
            result = export_hosts_task(
                output_file=output_file,
                target_id=123,  # 应该被忽略
                provider=provider
            )
            
            # 验证：使用了 provider
            assert result['success'] is True
            assert result['total_count'] == len(hosts)
            assert 'target_type' not in result  # Provider 模式不返回 target_type
            
            # 验证：文件内容正确
            with open(output_file, 'r') as f:
                lines = [line.strip() for line in f.readlines()]
                assert lines == hosts
        
        finally:
            Path(output_file).unlink(missing_ok=True)
    
    def test_error_when_no_parameters(self):
        """测试当 target_id 和 provider 都未提供时抛出错误"""
        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.txt') as f:
            output_file = f.name
        
        try:
            with pytest.raises(ValueError, match="必须提供 target_id 或 provider 参数之一"):
                export_hosts_task(output_file=output_file)
        
        finally:
            Path(output_file).unlink(missing_ok=True)


class TestExportSiteUrlsTaskBackwardCompatibility:
    """export_site_urls_task 向后兼容性测试"""
    
    def test_property_8_legacy_mode_uses_traditional_logic(self):
        """
        Property 8: Task Backward Compatibility (export_site_urls_task)
        
        Feature: scan-target-provider, Property 8: Task Backward Compatibility
        **Validates: Requirements 6.1, 6.2, 6.4, 6.5**
        
        When calling export_site_urls_task with only target_id,
        it should use the traditional logic (_export_site_urls_legacy).
        """
        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.txt') as f:
            output_file = f.name
        
        try:
            target_id = 123
            
            # Mock HostPortMappingService
            mock_associations = [
                {'host': 'example.com', 'port': 80},
                {'host': 'test.com', 'port': 443},
            ]
            
            with patch('apps.scan.tasks.site_scan.export_site_urls_task.HostPortMappingService') as mock_service_class, \
                 patch('apps.scan.tasks.site_scan.export_site_urls_task.BlacklistService') as mock_blacklist_service:
                
                # Mock HostPortMappingService
                mock_service = MagicMock()
                mock_service.iter_host_port_by_target.return_value = iter(mock_associations)
                mock_service_class.return_value = mock_service
                
                # Mock BlacklistService
                mock_blacklist = MagicMock()
                mock_blacklist.get_rules.return_value = []
                mock_blacklist_service.return_value = mock_blacklist
                
                # 调用任务（传统模式：只传 target_id）
                result = export_site_urls_task(
                    output_file=output_file,
                    target_id=target_id
                )
                
                # 验证：返回值包含传统模式的字段
                assert result['success'] is True
                assert result['output_file'] == output_file
                assert result['total_urls'] == 2  # 80 端口生成 1 个 URL，443 端口生成 1 个 URL
                assert 'association_count' in result  # 传统模式应该返回 association_count
                assert result['association_count'] == 2
                assert result['source'] == 'host_port'
                
                # 验证：文件内容正确
                with open(output_file, 'r') as f:
                    lines = [line.strip() for line in f.readlines()]
                    assert 'http://example.com' in lines
                    assert 'https://test.com' in lines
        
        finally:
            Path(output_file).unlink(missing_ok=True)
    
    def test_provider_mode_uses_provider_logic(self):
        """测试当提供 provider 时使用 Provider 模式"""
        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.txt') as f:
            output_file = f.name
        
        try:
            urls = ['https://example.com', 'https://test.com']
            provider = ListTargetProvider(targets=urls)
            
            # 调用任务（Provider 模式）
            result = export_site_urls_task(
                output_file=output_file,
                provider=provider
            )
            
            # 验证：使用了 provider
            assert result['success'] is True
            assert result['total_urls'] == len(urls)
            assert 'association_count' not in result  # Provider 模式不返回 association_count
            assert result['source'] == 'provider'
            
            # 验证：文件内容正确
            with open(output_file, 'r') as f:
                lines = [line.strip() for line in f.readlines()]
                assert lines == urls
        
        finally:
            Path(output_file).unlink(missing_ok=True)
    
    def test_error_when_no_parameters(self):
        """测试当 target_id 和 provider 都未提供时抛出错误"""
        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.txt') as f:
            output_file = f.name
        
        try:
            with pytest.raises(ValueError, match="必须提供 target_id 或 provider 参数之一"):
                export_site_urls_task(output_file=output_file)
        
        finally:
            Path(output_file).unlink(missing_ok=True)
