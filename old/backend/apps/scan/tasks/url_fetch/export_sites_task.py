"""
导出站点 URL 列表任务

使用 TargetProvider 从任意数据源导出 URL（用于 katana 等爬虫工具）。

数据源：WebSite，为空时回退到默认 URL
"""

import logging
from pathlib import Path


from apps.scan.providers import TargetProvider

logger = logging.getLogger(__name__)







def export_sites_task(
    output_file: str,
    provider: TargetProvider,
) -> dict:
    """
    导出站点 URL 列表到文件（用于 katana 等爬虫工具）

    数据源：WebSite，为空时回退到默认 URL

    Args:
        output_file: 输出文件路径
        provider: TargetProvider 实例

    Returns:
        dict: {
            'output_file': str,
            'asset_count': int,
            'source': str,  # website | default
        }

    Raises:
        ValueError: provider 未提供
    """
    if provider is None:
        raise ValueError("必须提供 provider 参数")

    logger.info("导出 URL - Provider: %s", type(provider).__name__)

    output_path = Path(output_file)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    # 按优先级获取数据源
    urls = list(provider.iter_websites())
    source = "website"

    if not urls:
        logger.info("WebSite 为空，生成默认 URL")
        urls = list(provider.iter_default_urls())
        source = "default"

    # 写入文件
    total_count = 0
    with open(output_path, 'w', encoding='utf-8', buffering=8192) as f:
        for url in urls:
            f.write(f"{url}\n")
            total_count += 1

    logger.info(
        "✓ URL 导出完成 - 来源: %s, 总数: %d, 文件: %s",
        source, total_count, str(output_path)
    )

    return {
        'output_file': str(output_path),
        'asset_count': total_count,
        'source': source,
    }
