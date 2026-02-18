"""初始化 Nuclei 模板仓库

项目安装后执行此命令，自动创建官方模板仓库记录。

使用方式：
    python manage.py init_nuclei_templates           # 只创建记录（检测本地已有仓库）
    python manage.py init_nuclei_templates --sync    # 创建并同步（git clone）
"""

import logging
import subprocess
from pathlib import Path

from django.conf import settings
from django.core.management.base import BaseCommand
from django.utils import timezone

from apps.engine.models import NucleiTemplateRepo
from apps.engine.services import NucleiTemplateRepoService

logger = logging.getLogger(__name__)


# 默认仓库配置（从 settings 读取，支持 Gitee 镜像）
DEFAULT_REPOS = [
    {
        "name": "nuclei-templates",
        "repo_url": getattr(settings, 'NUCLEI_TEMPLATES_REPO_URL', 'https://github.com/projectdiscovery/nuclei-templates.git'),
        "description": "Nuclei 官方模板仓库，包含数千个漏洞检测模板",
    },
]


def get_local_commit_hash(local_path: Path) -> str:
    """获取本地 Git 仓库的 commit hash"""
    if not (local_path / ".git").is_dir():
        return ""
    result = subprocess.run(
        ["git", "-C", str(local_path), "rev-parse", "HEAD"],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    return result.stdout.strip() if result.returncode == 0 else ""


class Command(BaseCommand):
    help = "初始化 Nuclei 模板仓库（创建官方模板仓库记录）"

    def add_arguments(self, parser):
        parser.add_argument(
            "--sync",
            action="store_true",
            help="创建后立即同步（git clone），首次需要较长时间",
        )
        parser.add_argument(
            "--force",
            action="store_true",
            help="强制重新创建（删除已存在的同名仓库）",
        )

    def handle(self, *args, **options):
        do_sync = options.get("sync", False)
        force = options.get("force", False)

        service = NucleiTemplateRepoService()
        base_dir = Path(getattr(settings, "NUCLEI_TEMPLATES_REPOS_BASE_DIR", "/opt/xingrin/nuclei-repos"))
        
        created = 0
        skipped = 0
        synced = 0

        for repo_config in DEFAULT_REPOS:
            name = repo_config["name"]
            repo_url = repo_config["repo_url"]

            # 检查是否已存在
            existing = NucleiTemplateRepo.objects.filter(name=name).first()

            if existing:
                if force:
                    self.stdout.write(self.style.WARNING(
                        f"[{name}] 强制模式，删除已存在的仓库记录"
                    ))
                    service.remove_local_path_dir(existing)
                    existing.delete()
                else:
                    self.stdout.write(self.style.SUCCESS(
                        f"[{name}] 已存在，跳过创建"
                    ))
                    skipped += 1

                    # 如果需要同步且已存在，也执行同步
                    if do_sync and existing.id:
                        try:
                            result = service.refresh_repo(existing.id)
                            self.stdout.write(self.style.SUCCESS(
                                f"[{name}] 同步完成: {result.get('action', 'unknown')}, "
                                f"commit={result.get('commitHash', 'N/A')[:8]}"
                            ))
                            synced += 1
                        except Exception as e:
                            self.stdout.write(self.style.ERROR(
                                f"[{name}] 同步失败: {e}"
                            ))
                    continue

            # 创建新仓库记录
            try:
                # 检查本地是否已有仓库（由 install.sh 预下载）
                local_path = base_dir / name
                local_commit = get_local_commit_hash(local_path)
                
                repo = NucleiTemplateRepo.objects.create(
                    name=name,
                    repo_url=repo_url,
                    local_path=str(local_path) if local_commit else "",
                    commit_hash=local_commit,
                    last_synced_at=timezone.now() if local_commit else None,
                )
                
                if local_commit:
                    self.stdout.write(self.style.SUCCESS(
                        f"[{name}] 创建成功（检测到本地仓库）: commit={local_commit[:8]}"
                    ))
                else:
                    self.stdout.write(self.style.SUCCESS(
                        f"[{name}] 创建成功: id={repo.id}"
                    ))
                created += 1

                # 如果本地没有仓库且需要同步
                if not local_commit and do_sync:
                    try:
                        self.stdout.write(self.style.WARNING(
                            f"[{name}] 正在同步（首次可能需要几分钟）..."
                        ))
                        result = service.refresh_repo(repo.id)
                        self.stdout.write(self.style.SUCCESS(
                            f"[{name}] 同步完成: {result.get('action', 'unknown')}, "
                            f"commit={result.get('commitHash', 'N/A')[:8]}"
                        ))
                        synced += 1
                    except Exception as e:
                        self.stdout.write(self.style.ERROR(
                            f"[{name}] 同步失败: {e}"
                        ))

            except Exception as e:
                self.stdout.write(self.style.ERROR(
                    f"[{name}] 创建失败: {e}"
                ))

        self.stdout.write(self.style.SUCCESS(
            f"\n初始化完成: 创建 {created}, 跳过 {skipped}, 同步 {synced}"
        ))
