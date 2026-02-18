#!/usr/bin/env python
"""
扫描任务启动脚本

用于动态扫描容器启动时执行。
必须在 Django 导入之前获取配置并设置环境变量。
"""
import argparse
import sys
import os
import traceback


def main():
    print("="*60)
    print("run_initiate_scan.py 启动")
    print(f"  Python: {sys.version}")
    print(f"  CWD: {os.getcwd()}")
    print(f"  SERVER_URL: {os.environ.get('SERVER_URL', 'NOT SET')}")
    print("="*60)
    
    # 1. 从配置中心获取配置并初始化 Django（必须在 Django 导入之前）
    print("[1/4] 从配置中心获取配置...")
    try:
        from apps.common.container_bootstrap import fetch_config_and_setup_django
        fetch_config_and_setup_django()
        print("[1/4] ✓ 配置获取成功")
    except Exception as e:
        print(f"[1/4] ✗ 配置获取失败: {e}")
        traceback.print_exc()
        sys.exit(1)
    
    # 2. 解析命令行参数
    print("[2/4] 解析命令行参数...")
    parser = argparse.ArgumentParser(description="执行扫描初始化 Flow")
    parser.add_argument("--scan_id", type=int, required=True, help="扫描任务 ID")
    parser.add_argument("--target_id", type=int, required=True, help="目标 ID")
    parser.add_argument("--scan_workspace_dir", type=str, required=True, help="扫描工作目录")
    parser.add_argument("--engine_name", type=str, required=True, help="引擎名称")
    parser.add_argument("--scheduled_scan_name", type=str, default=None, help="定时扫描任务名称（可选）")

    args = parser.parse_args()
    print("[2/4] ✓ 参数解析成功:")
    print(f"       scan_id: {args.scan_id}")
    print(f"       target_id: {args.target_id}")
    print(f"       scan_workspace_dir: {args.scan_workspace_dir}")
    print(f"       engine_name: {args.engine_name}")
    print(f"       scheduled_scan_name: {args.scheduled_scan_name}")
    
    # 3. 现在可以安全导入 Django 相关模块
    print("[3/4] 导入 initiate_scan_flow...")
    try:
        from apps.scan.flows.initiate_scan_flow import initiate_scan_flow
        print("[3/4] ✓ 导入成功")
    except Exception as e:
        print(f"[3/4] ✗ 导入失败: {e}")
        traceback.print_exc()
        sys.exit(1)
    
    # 4. 执行 Flow
    print("[4/4] 执行 initiate_scan_flow...")
    try:
        result = initiate_scan_flow(
            scan_id=args.scan_id,
            target_id=args.target_id,
            scan_workspace_dir=args.scan_workspace_dir,
            engine_name=args.engine_name,
            scheduled_scan_name=args.scheduled_scan_name,
        )
        print("[4/4] ✓ Flow 执行完成")
        print(f"结果: {result}")
    except Exception as e:
        print(f"[4/4] ✗ Flow 执行失败: {e}")
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
