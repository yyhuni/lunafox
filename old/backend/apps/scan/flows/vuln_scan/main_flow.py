"""
漏洞扫描主 Flow
"""
import logging
from typing import Dict, Tuple

from apps.scan.decorators import scan_flow
from apps.scan.handlers.scan_flow_handlers import (
    on_scan_flow_running,
    on_scan_flow_completed,
    on_scan_flow_failed,
)
from apps.scan.configs.command_templates import get_command_template
from apps.scan.utils import user_log, wait_for_system_load
from .endpoints_vuln_scan_flow import endpoints_vuln_scan_flow
from .websites_vuln_scan_flow import websites_vuln_scan_flow


logger = logging.getLogger(__name__)


def _classify_vuln_tools(
    enabled_tools: Dict[str, dict]
) -> Tuple[Dict[str, dict], Dict[str, dict], Dict[str, dict]]:
    """根据用户配置分类漏洞扫描工具。

    分类逻辑：
    - 读取 scan_endpoints / scan_websites 配置
    - 默认值从模板的 defaults 或 input_type 推断

    Returns:
        (endpoints_tools, websites_tools, other_tools) 三元组
    """
    endpoints_tools: Dict[str, dict] = {}
    websites_tools: Dict[str, dict] = {}
    other_tools: Dict[str, dict] = {}

    for tool_name, tool_config in enabled_tools.items():
        template = get_command_template("vuln_scan", tool_name) or {}
        defaults = template.get("defaults", {})

        # 根据 input_type 推断默认值（兼容老工具）
        input_type = template.get("input_type")
        default_endpoints = defaults.get("scan_endpoints", input_type == "endpoints_file")
        default_websites = defaults.get("scan_websites", input_type == "websites_file")

        scan_endpoints = tool_config.get("scan_endpoints", default_endpoints)
        scan_websites = tool_config.get("scan_websites", default_websites)

        if scan_endpoints:
            endpoints_tools[tool_name] = tool_config
        if scan_websites:
            websites_tools[tool_name] = tool_config
        if not scan_endpoints and not scan_websites:
            other_tools[tool_name] = tool_config

    return endpoints_tools, websites_tools, other_tools


@scan_flow(
    name="vuln_scan",
    on_running=[on_scan_flow_running],
    on_completion=[on_scan_flow_completed],
    on_failure=[on_scan_flow_failed],
)
def vuln_scan_flow(
    scan_id: int,
    target_id: int,
    scan_workspace_dir: str,
    enabled_tools: Dict[str, dict],
    provider,
) -> dict:
    """漏洞扫描主 Flow：串行编排各类漏洞扫描子 Flow。

    支持工具：
    - dalfox_xss: XSS 漏洞扫描（流式保存）
    - nuclei: 通用漏洞扫描（流式保存，支持 endpoints 和 websites 两种输入）
    """
    try:
        # 负载检查：等待系统资源充足
        wait_for_system_load(context="vuln_scan_flow")

        # 从 provider 获取 target_name
        target_name = provider.get_target_name()
        if not target_name:
            raise ValueError("无法获取 Target 名称")

        if scan_id is None:
            raise ValueError("scan_id 不能为空")
        if target_id is None:
            raise ValueError("target_id 不能为空")
        if not scan_workspace_dir:
            raise ValueError("scan_workspace_dir 不能为空")
        if not enabled_tools:
            raise ValueError("enabled_tools 不能为空")

        logger.info("开始漏洞扫描 - Scan ID: %s, Target: %s", scan_id, target_name)
        user_log(scan_id, "vuln_scan", "Starting vulnerability scan")

        # Step 1: 分类工具
        endpoints_tools, websites_tools, other_tools = _classify_vuln_tools(enabled_tools)

        logger.info(
            "漏洞扫描工具分类 - endpoints: %s, websites: %s, 其他: %s",
            list(endpoints_tools.keys()) or "无",
            list(websites_tools.keys()) or "无",
            list(other_tools.keys()) or "无",
        )

        if other_tools:
            logger.warning(
                "存在暂不支持输入类型的漏洞扫描工具，将被忽略: %s",
                list(other_tools.keys()),
            )

        if not endpoints_tools and not websites_tools:
            raise ValueError(
                "漏洞扫描需要至少启用一个工具（endpoints 或 websites 模式）"
            )

        total_vulns = 0
        results = {}

        # Step 2: 执行 Endpoint 漏洞扫描子 Flow
        if endpoints_tools:
            logger.info("执行 Endpoint 漏洞扫描 - 工具: %s", list(endpoints_tools.keys()))
            endpoint_result = endpoints_vuln_scan_flow(
                scan_id=scan_id,
                target_id=target_id,
                scan_workspace_dir=scan_workspace_dir,
                enabled_tools=endpoints_tools,
                provider=provider,
            )
            results["endpoints"] = endpoint_result
            total_vulns += sum(
                r.get("created_vulns", 0)
                for r in endpoint_result.get("tool_results", {}).values()
            )

        # Step 3: 执行 WebSite 漏洞扫描子 Flow
        if websites_tools:
            logger.info("执行 WebSite 漏洞扫描 - 工具: %s", list(websites_tools.keys()))
            website_result = websites_vuln_scan_flow(
                scan_id=scan_id,
                target_id=target_id,
                scan_workspace_dir=scan_workspace_dir,
                enabled_tools=websites_tools,
                provider=provider,
            )
            results["websites"] = website_result
            total_vulns += sum(
                r.get("created_vulns", 0)
                for r in website_result.get("tool_results", {}).values()
            )

        # 记录 Flow 完成
        logger.info("✓ 漏洞扫描完成 - 新增漏洞: %d", total_vulns)
        user_log(scan_id, "vuln_scan", f"vuln_scan completed: found {total_vulns} vulnerabilities")

        return {
            "success": True,
            "scan_id": scan_id,
            "target": target_name,
            "scan_workspace_dir": scan_workspace_dir,
            "total_vulns": total_vulns,
            "sub_flow_results": results,
        }

    except Exception as e:
        logger.exception("漏洞扫描主 Flow 失败: %s", e)
        raise
