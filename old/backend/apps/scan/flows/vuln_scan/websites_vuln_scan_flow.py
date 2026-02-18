"""
基于 WebSite 的漏洞扫描 Flow

与 endpoints_vuln_scan_flow 类似，但数据源是 WebSite 而不是 Endpoint。
主要用于 nuclei 扫描已存活的网站。
"""

import logging
from datetime import datetime
from typing import Dict

from concurrent.futures import ThreadPoolExecutor

from apps.scan.decorators import scan_flow
from apps.scan.utils import build_scan_command, ensure_nuclei_templates_local, user_log
from apps.scan.tasks.vuln_scan import run_and_stream_save_nuclei_vulns_task
from apps.scan.tasks.vuln_scan.export_websites_task import export_websites_task
from .utils import calculate_timeout_by_line_count

logger = logging.getLogger(__name__)


@scan_flow(name="websites_vuln_scan_flow")
def websites_vuln_scan_flow(
    scan_id: int,
    target_id: int,
    scan_workspace_dir: str,
    enabled_tools: Dict[str, dict],
    provider,
) -> dict:
    """基于 WebSite 的漏洞扫描 Flow（主要用于 nuclei）。"""
    try:
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

        from apps.scan.utils import setup_scan_directory
        vuln_scan_dir = setup_scan_directory(scan_workspace_dir, 'vuln_scan')
        websites_file = vuln_scan_dir / "input_websites.txt"

        # Step 1: 导出 WebSite URL
        export_result = export_websites_task(
            output_file=str(websites_file),
            provider=provider,
        )
        total_websites = export_result.get("total_count", 0)

        if total_websites == 0:
            logger.warning("目标下没有可用 WebSite，跳过漏洞扫描")
            return {
                "success": True,
                "scan_id": scan_id,
                "target": target_name,
                "scan_workspace_dir": scan_workspace_dir,
                "websites_file": str(websites_file),
                "website_count": 0,
                "executed_tools": [],
                "tool_results": {},
            }

        logger.info("WebSite 导出完成，共 %d 条，开始执行漏洞扫描", total_websites)

        tool_results: Dict[str, dict] = {}
        tool_futures: Dict[str, dict] = {}

        # Step 2: 执行漏洞扫描工具
        for tool_name, tool_config in enabled_tools.items():
            # 目前只支持 nuclei
            if tool_name != "nuclei":
                logger.warning("websites_vuln_scan_flow 暂不支持工具: %s", tool_name)
                continue

            # 确保 nuclei 模板存在
            repo_names = tool_config.get("template_repo_names")
            if not repo_names or not isinstance(repo_names, (list, tuple)):
                logger.error("Nuclei 配置缺少 template_repo_names（数组），跳过")
                continue

            template_paths = []
            try:
                for repo_name in repo_names:
                    path = ensure_nuclei_templates_local(repo_name)
                    template_paths.append(path)
                    logger.info("Nuclei 模板路径 [%s]: %s", repo_name, path)
            except Exception as e:
                logger.error("获取 Nuclei 模板失败: %s，跳过 nuclei 扫描", e)
                continue

            template_args = " ".join(f"-t {p}" for p in template_paths)

            # 构建命令（使用 websites_file 作为输入）
            command_params = {
                "input_file": str(websites_file),
                "template_args": template_args,
            }

            command = build_scan_command(
                tool_name=tool_name,
                scan_type="vuln_scan",
                command_params=command_params,
                tool_config=tool_config,
            )

            # 计算超时时间
            raw_timeout = tool_config.get("timeout", 600)
            if isinstance(raw_timeout, str) and raw_timeout == "auto":
                timeout = calculate_timeout_by_line_count(
                    tool_config=tool_config,
                    file_path=str(websites_file),
                    base_per_time=30,
                )
            else:
                try:
                    timeout = int(raw_timeout)
                except (TypeError, ValueError) as e:
                    raise ValueError(
                        f"工具 {tool_name} 的 timeout 配置无效: {raw_timeout!r}"
                    ) from e

            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            log_file = vuln_scan_dir / f"{tool_name}_websites_{timestamp}.log"

            logger.info("开始执行 %s 漏洞扫描（WebSite 模式）", tool_name)
            user_log(scan_id, "vuln_scan", f"Running {tool_name} (websites): {command}")

            tool_futures[tool_name] = {
                "command": command,
                "timeout": timeout,
                "log_file": str(log_file),
            }

        # 使用 ThreadPoolExecutor 并行执行
        if tool_futures:
            with ThreadPoolExecutor(max_workers=len(tool_futures)) as executor:
                futures = {}
                for tool_name, meta in tool_futures.items():
                    future = executor.submit(
                        run_and_stream_save_nuclei_vulns_task,
                        cmd=meta["command"],
                        tool_name=tool_name,
                        scan_id=scan_id,
                        target_id=target_id,
                        cwd=str(vuln_scan_dir),
                        shell=True,
                        batch_size=1,
                        timeout=meta["timeout"],
                        log_file=meta["log_file"],
                    )
                    futures[tool_name] = future

                # 收集结果
                for tool_name, future in futures.items():
                    meta = tool_futures[tool_name]
                    try:
                        result = future.result()
                        created_vulns = result.get("created_vulns", 0)
                        tool_results[tool_name] = {
                            "command": meta["command"],
                            "timeout": meta["timeout"],
                            "processed_records": result.get("processed_records"),
                            "created_vulns": created_vulns,
                            "command_log_file": meta["log_file"],
                        }
                        logger.info(
                            "✓ 工具 %s (websites) 执行完成 - 漏洞: %d",
                            tool_name, created_vulns
                        )
                        user_log(
                            scan_id, "vuln_scan",
                            f"{tool_name} (websites) completed: found {created_vulns} vulnerabilities"
                        )
                    except Exception as e:
                        reason = str(e)
                        logger.error("工具 %s 执行失败: %s", tool_name, e, exc_info=True)
                        user_log(scan_id, "vuln_scan", f"{tool_name} failed: {reason}", "error")

        return {
            "success": True,
            "scan_id": scan_id,
            "target": target_name,
            "scan_workspace_dir": scan_workspace_dir,
            "websites_file": str(websites_file),
            "website_count": total_websites,
            "executed_tools": list(enabled_tools.keys()),
            "tool_results": tool_results,
        }

    except Exception as e:
        logger.exception("WebSite 漏洞扫描失败: %s", e)
        raise
