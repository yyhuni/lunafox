"""
URL Fetch 共享工具函数
"""

import logging
import subprocess
import uuid
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from pathlib import Path

from apps.scan.utils import build_scan_command

logger = logging.getLogger(__name__)


def calculate_timeout_by_line_count(
    tool_config: dict,
    file_path: str,
    base_per_time: int = 1,
    min_timeout: int = 60,
) -> int:
    """
    根据文件行数自动计算超时时间

    Args:
        tool_config: 工具配置（保留参数，未来可能用于更复杂的计算）
        file_path: 输入文件路径
        base_per_time: 每行的基础时间（秒）
        min_timeout: 最小超时时间（秒），默认60秒

    Returns:
        int: 计算出的超时时间（秒），不低于 min_timeout
    """
    try:
        result = subprocess.run(
            ['wc', '-l', file_path],
            capture_output=True,
            text=True,
            check=True,
        )
        line_count = int(result.stdout.strip().split()[0])
        timeout = max(line_count * base_per_time, min_timeout)
        logger.info(
            "timeout 自动计算: 文件=%s, 行数=%d, 每行时间=%d秒, 最小值=%d秒, timeout=%d秒",
            file_path,
            line_count,
            base_per_time,
            min_timeout,
            timeout,
        )
        return timeout
    except Exception as e:
        logger.warning("wc -l 计算行数失败: %s，将使用默认 timeout: %d秒", e, min_timeout)
        return min_timeout


def prepare_tool_execution(
    tool_name: str,
    tool_config: dict,
    input_file: str,
    input_type: str,
    output_dir: Path,
    scan_type: str = "url_fetch"
) -> dict:
    """
    准备单个工具的执行参数

    Args:
        tool_name: 工具名称
        tool_config: 工具配置
        input_file: 输入文件路径
        input_type: 输入类型（domains_file 或 sites_file）
        output_dir: 输出目录
        scan_type: 扫描类型

    Returns:
        dict: 执行参数，包含 command, input_file, output_file, timeout
              或包含 error 键表示失败
    """
    # 1. 统计输入文件行数
    try:
        with open(input_file, 'r') as f:
            input_count = sum(1 for _ in f)
        logger.info("工具 %s - 输入类型: %s, 数量: %d", tool_name, input_type, input_count)
    except Exception as e:
        return {"error": f"读取输入文件失败: {e}"}

    # 2. 生成输出文件路径（带时间戳和短 UUID 后缀）
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    short_uuid = uuid.uuid4().hex[:4]
    output_file = str(output_dir / f"{tool_name}_{timestamp}_{short_uuid}.txt")

    # 3. 构建命令
    command_params = {
        input_type: input_file,
        "output_file": output_file,
    }

    try:
        command = build_scan_command(
            tool_name=tool_name,
            scan_type=scan_type,
            command_params=command_params,
            tool_config=tool_config,
        )
    except Exception as e:
        logger.error("构建 %s 命令失败: %s", tool_name, e)
        return {"error": f"命令构建失败: {e}"}

    # 4. 计算超时时间（支持 auto 和显式整数）
    raw_timeout = tool_config.get("timeout", 3600)
    timeout = 3600

    if isinstance(raw_timeout, str) and raw_timeout == "auto":
        try:
            # katana / waymore 每个站点需要更长时间
            base_per_time = 360 if tool_name in ("katana", "waymore") else 1
            timeout = calculate_timeout_by_line_count(
                tool_config=tool_config,
                file_path=input_file,
                base_per_time=base_per_time,
            )
        except Exception as e:
            logger.warning(
                "工具 %s 自动计算 timeout 失败，将使用默认 3600 秒: %s",
                tool_name,
                e,
            )
            timeout = 3600
    else:
        try:
            timeout = int(raw_timeout)
        except (TypeError, ValueError):
            logger.warning(
                "工具 %s 的 timeout 配置无效(%s)，将使用默认 3600 秒",
                tool_name,
                raw_timeout,
            )
            timeout = 3600

    # 5. 返回执行参数
    return {
        "command": command,
        "input_file": input_file,
        "input_type": input_type,
        "output_file": output_file,
        "timeout": timeout,
    }


def run_tools_parallel(
    tools: dict,
    input_file: str,
    input_type: str,
    output_dir: Path,
    scan_id: int
) -> tuple[list, list, list]:
    """
    并行执行工具列表

    Args:
        tools: 工具配置字典 {tool_name: tool_config}
        input_file: 输入文件路径
        input_type: 输入类型
        output_dir: 输出目录
        scan_id: 扫描任务 ID（用于记录日志）

    Returns:
        tuple: (result_files, failed_tools, successful_tool_names)
    """
    from apps.scan.tasks.url_fetch import run_url_fetcher_task
    from apps.scan.utils import user_log

    tool_params = {}  # 存储每个工具的参数
    failed_tools: list[dict] = []

    # 准备所有工具的参数
    for tool_name, tool_config in tools.items():
        exec_params = prepare_tool_execution(
            tool_name=tool_name,
            tool_config=tool_config,
            input_file=input_file,
            input_type=input_type,
            output_dir=output_dir,
        )

        if "error" in exec_params:
            failed_tools.append({"tool": tool_name, "reason": exec_params["error"]})
            continue

        logger.info(
            "提交任务 - 工具: %s, 输入: %s, 超时: %d秒",
            tool_name,
            input_type,
            exec_params["timeout"],
        )

        # 记录工具开始执行日志
        user_log(scan_id, "url_fetch", f"Running {tool_name}: {exec_params['command']}")

        tool_params[tool_name] = exec_params

    # 使用 ThreadPoolExecutor 并行执行
    result_files = []
    if tool_params:
        with ThreadPoolExecutor(max_workers=len(tool_params)) as executor:
            futures = {}
            for tool_name, params in tool_params.items():
                future = executor.submit(
                    run_url_fetcher_task,
                    tool_name=tool_name,
                    command=params["command"],
                    timeout=params["timeout"],
                    output_file=params["output_file"],
                )
                futures[tool_name] = future

            # 收集执行结果
            for tool_name, future in futures.items():
                try:
                    result = future.result()
                    if result and result['success']:
                        result_files.append(result['output_file'])
                        url_count = result['url_count']
                        logger.info(
                            "✓ 工具 %s 执行成功 - 发现 URL: %d",
                            tool_name, url_count
                        )
                        user_log(
                            scan_id, "url_fetch",
                            f"{tool_name} completed: found {url_count} urls"
                        )
                    else:
                        reason = '未生成结果或无有效URL'
                        failed_tools.append({'tool': tool_name, 'reason': reason})
                        logger.warning("⚠️ 工具 %s 未生成有效结果", tool_name)
                        user_log(
                            scan_id, "url_fetch",
                            f"{tool_name} failed: {reason}", "error"
                        )
                except Exception as e:
                    reason = str(e)
                    failed_tools.append({'tool': tool_name, 'reason': reason})
                    logger.warning("⚠️ 工具 %s 执行失败: %s", tool_name, e)
                    user_log(
                        scan_id, "url_fetch",
                        f"{tool_name} failed: {reason}", "error"
                    )

    # 计算成功的工具列表
    failed_tool_names = [f['tool'] for f in failed_tools]
    successful_tool_names = [
        name for name in tools.keys()
        if name not in failed_tool_names
    ]

    return result_files, failed_tools, successful_tool_names
