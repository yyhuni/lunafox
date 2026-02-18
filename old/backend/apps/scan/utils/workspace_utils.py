"""
工作空间工具模块

提供统一的扫描工作目录创建和验证功能
"""

from pathlib import Path
import logging

logger = logging.getLogger(__name__)


def setup_scan_workspace(scan_workspace_dir: str) -> Path:
    """
    创建 Scan 根工作空间目录
    
    Args:
        scan_workspace_dir: 工作空间目录路径
        
    Returns:
        Path: 创建的目录路径
        
    Raises:
        RuntimeError: 目录创建失败或不可写
    """
    workspace_path = Path(scan_workspace_dir)
    
    try:
        workspace_path.mkdir(parents=True, exist_ok=True)
    except OSError as e:
        raise RuntimeError(f"创建工作空间失败: {scan_workspace_dir} - {e}") from e
    
    # 验证可写
    _verify_writable(workspace_path)
    
    logger.info("✓ Scan 工作空间已创建: %s", workspace_path)
    return workspace_path


def setup_scan_directory(scan_workspace_dir: str, subdir: str) -> Path:
    """
    创建扫描子目录
    
    Args:
        scan_workspace_dir: 根工作空间目录
        subdir: 子目录名称（如 'fingerprint_detect', 'site_scan'）
        
    Returns:
        Path: 创建的子目录路径
        
    Raises:
        RuntimeError: 目录创建失败或不可写
    """
    scan_dir = Path(scan_workspace_dir) / subdir
    
    try:
        scan_dir.mkdir(parents=True, exist_ok=True)
    except OSError as e:
        raise RuntimeError(f"创建扫描目录失败: {scan_dir} - {e}") from e
    
    # 验证可写
    _verify_writable(scan_dir)
    
    logger.info("✓ 扫描目录已创建: %s", scan_dir)
    return scan_dir


def _verify_writable(path: Path) -> None:
    """
    验证目录可写
    
    Args:
        path: 目录路径
        
    Raises:
        RuntimeError: 目录不可写
    """
    test_file = path / ".test_write"
    try:
        test_file.touch()
        test_file.unlink()
    except OSError as e:
        raise RuntimeError(f"目录不可写: {path} - {e}") from e
