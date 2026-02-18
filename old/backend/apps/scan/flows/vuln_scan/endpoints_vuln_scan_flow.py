"""
基于 Endpoint 的漏洞扫描 Flow
"""

import logging
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from typing import Dict

from apps.scan.decorators import scan_flow
from apps.scan.utils import build_scan_command, ensure_nuclei_templates_local, user_log
from apps.scan.tasks.vuln_scan import (
    export_endpoints_task,
    run_vuln_tool_task,
    run_and_stream_save_dalfox_vulns_task,
    run_and_stream_save_nuclei_vulns_task,
)
from .utils import calculate_timeout_by_line_count


logger = logging.getLogger(__name__)


@scan_flow(name="endpoints_vuln_scan_flow")
def endpoints_vuln_scan_flow(
    scan_id: int,
    target_id: int,
    scan_workspace_dir: str,
    enabled_tools: Dict[str, dict],
    provider,
) -> dict:
    """基于 Endpoint 的漏洞扫描 Flow（串行执行 Dalfox 等工具）。"""
    try:
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

        from apps.scan.utils import setup_scan_directory
        vuln_scan_dir = setup_scan_directory(scan_workspace_dir, 'vuln_scan')
        endpoints_file = vuln_scan_dir / "input_endpoints.txt"

        # Step 1: 导出 Endpoint URL
        export_result = export_endpoints_task(
            output_file=str(endpoints_file),
            provider=provider,
        )
        total_endpoints = export_result.get("total_count", 0)

        if total_endpoints == 0 or not endpoints_file.exists() or endpoints_file.stat().st_size == 0:
            logger.warning("目标下没有可用 Endpoint，跳过漏洞扫描")
            return {
                "success": True,
                "scan_id": scan_id,
                "target": target_name,
                "scan_workspace_dir": scan_workspace_dir,
                "endpoints_file": str(endpoints_file),
                "endpoint_count": 0,
                "executed_tools": [],
                "tool_results": {},
            }

        logger.info("Endpoint 导出完成，共 %d 条，开始执行漏洞扫描", total_endpoints)

        tool_results: Dict[str, dict] = {}
        tool_params: Dict[str, dict] = {}  # 存储每个工具的参数

        # Step 2: 准备每个漏洞扫描工具的参数
        for tool_name, tool_config in enabled_tools.items():
            # Nuclei 需要先确保本地模板存在（支持多个模板仓库）
            template_args = ""
            if tool_name == "nuclei":
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

            # 构建命令参数（根据工具模板使用不同的参数名）
            if tool_name == "nuclei":
                command_params = {"input_file": str(endpoints_file)}
            else:
                command_params = {"endpoints_file": str(endpoints_file)}
            if template_args:
                command_params["template_args"] = template_args

            command = build_scan_command(
                tool_name=tool_name,
                scan_type="vuln_scan",
                command_params=command_params,
                tool_config=tool_config,
            )

            raw_timeout = tool_config.get("timeout", 600)

            if isinstance(raw_timeout, str) and raw_timeout == "auto":
                # timeout=auto 时，根据 endpoints_file 行数自动计算超时时间
                # Dalfox: 每行 100 秒，Nuclei: 每行 30 秒
                base_per_time = 30 if tool_name == "nuclei" else 100
                timeout = calculate_timeout_by_line_count(
                    tool_config=tool_config,
                    file_path=str(endpoints_file),
                    base_per_time=base_per_time,
                )
            else:
                try:
                    timeout = int(raw_timeout)
                except (TypeError, ValueError) as e:
                    raise ValueError(
                        f"工具 {tool_name} 的 timeout 配置无效: {raw_timeout!r}"
                    ) from e

            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            log_file = vuln_scan_dir / f"{tool_name}_{timestamp}.log"

            logger.info("开始执行漏洞扫描工具 %s", tool_name)
            user_log(scan_id, "vuln_scan", f"Running {tool_name}: {command}")

            # 确定工具类型
            if tool_name == "dalfox_xss":
                mode = "dalfox"
            elif tool_name == "nuclei":
                mode = "nuclei"
            else:
                mode = "normal"

            tool_params[tool_name] = {
                "command": command,
                "timeout": timeout,
                "log_file": str(log_file),
                "mode": mode,
            }

        # Step 3: 使用 ThreadPoolExecutor 并行执行
        if tool_params:
            with ThreadPoolExecutor(max_workers=len(tool_params)) as executor:
                futures = {}
                for tool_name, params in tool_params.items():
                    if params["mode"] == "dalfox":
                        future = executor.submit(
                            run_and_stream_save_dalfox_vulns_task,
                            cmd=params["command"],
                            tool_name=tool_name,
                            scan_id=scan_id,
                            target_id=target_id,
                            cwd=str(vuln_scan_dir),
                            shell=True,
                            batch_size=1,
                            timeout=params["timeout"],
                            log_file=params["log_file"],
                        )
                    elif params["mode"] == "nuclei":
                        future = executor.submit(
                            run_and_stream_save_nuclei_vulns_task,
                            cmd=params["command"],
                            tool_name=tool_name,
                            scan_id=scan_id,
                            target_id=target_id,
                            cwd=str(vuln_scan_dir),
                            shell=True,
                            batch_size=1,
                            timeout=params["timeout"],
                            log_file=params["log_file"],
                        )
                    else:
                        future = executor.submit(
                            run_vuln_tool_task,
                            tool_name=tool_name,
                            command=params["command"],
                            timeout=params["timeout"],
                            log_file=params["log_file"],
                        )
                    futures[tool_name] = future

                # 收集结果
                for tool_name, future in futures.items():
                    params = tool_params[tool_name]
                    try:
                        result = future.result()

                        if params["mode"] in ("dalfox", "nuclei"):
                            created_vulns = result.get("created_vulns", 0)
                            tool_results[tool_name] = {
                                "command": params["command"],
                                "timeout": params["timeout"],
                                "processed_records": result.get("processed_records"),
                                "created_vulns": created_vulns,
                                "command_log_file": params["log_file"],
                            }
                            logger.info(
                                "✓ 工具 %s 执行完成 - 漏洞: %d",
                                tool_name, created_vulns
                            )
                            user_log(
                                scan_id, "vuln_scan",
                                f"{tool_name} completed: found {created_vulns} vulnerabilities"
                            )
                        else:
                            tool_results[tool_name] = {
                                "command": params["command"],
                                "timeout": params["timeout"],
                                "duration": result.get("duration"),
                                "returncode": result.get("returncode"),
                                "command_log_file": result.get("command_log_file"),
                            }
                            logger.info(
                                "✓ 工具 %s 执行完成 - returncode=%s",
                                tool_name, result.get("returncode")
                            )
                            user_log(scan_id, "vuln_scan", f"{tool_name} completed")
                    except Exception as e:
                        reason = str(e)
                        logger.error("工具 %s 执行失败: %s", tool_name, e, exc_info=True)
                        user_log(scan_id, "vuln_scan", f"{tool_name} failed: {reason}", "error")

        return {
            "success": True,
            "scan_id": scan_id,
            "target": target_name,
            "scan_workspace_dir": scan_workspace_dir,
            "endpoints_file": str(endpoints_file),
            "endpoint_count": total_endpoints,
            "executed_tools": list(enabled_tools.keys()),
            "tool_results": tool_results,
        }

    except Exception as e:
        logger.exception("Endpoint 漏洞扫描失败: %s", e)
        raise
