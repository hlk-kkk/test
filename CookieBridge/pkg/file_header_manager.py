# -*- coding: utf-8 -*-
import os
import re
import sys
from typing import List, Tuple

# 项目配置
REPO_URL = "https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge"
GITHUB_PROFILE = "https://github.com/NanmiCoder"
EMAIL = "relakkes@gmail.com"
COPYRIGHT_YEAR = "2025"
LICENSE_TYPE = "NON-COMMERCIAL LEARNING LICENSE 1.1"
PROJECT_NAME = "MediaCrawlerPro-CookieBridge"

# 免责声明（Python 注释风格）
DISCLAIMER_PY = """# 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
# 1. 不得用于任何商业用途。
# 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
# 3. 不得进行大规模爬取或对平台造成运营干扰。
# 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
# 5. 不得用于任何非法或不当的用途。
#
# 详细许可条款请参阅项目根目录下的LICENSE文件。
# 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。"""

# 免责声明（JS 注释风格）
DISCLAIMER_JS = """// 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
// 1. 不得用于任何商业用途。
// 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
// 3. 不得进行大规模爬取或对平台造成运营干扰。
// 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
// 5. 不得用于任何非法或不当的用途。
//
// 详细许可条款请参阅项目根目录下的LICENSE文件。
// 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。"""

EXCLUDE_DIRS = ['venv', '.venv', 'node_modules', '__pycache__', '.git', 'build', 'dist', '.eggs']


def get_file_relative_path(file_path: str, project_root: str) -> str:
    return os.path.relpath(file_path, project_root)


def detect_file_type(file_path: str) -> str:
    ext = os.path.splitext(file_path)[1].lower()
    if ext == '.py':
        return 'python'
    elif ext in ('.js', '.ts', '.jsx', '.tsx'):
        return 'javascript'
    return 'unknown'


def generate_copyright_header_py(relative_path: str) -> str:
    file_url = f"{REPO_URL}/blob/main/{relative_path}"
    return f"""# Copyright (c) {COPYRIGHT_YEAR} {EMAIL}
#
# This file is part of {PROJECT_NAME} project.
# Repository: {file_url}
# GitHub: {GITHUB_PROFILE}
# Licensed under {LICENSE_TYPE}
#"""


def generate_copyright_header_js(relative_path: str) -> str:
    file_url = f"{REPO_URL}/blob/main/{relative_path}"
    return f"""// Copyright (c) {COPYRIGHT_YEAR} {EMAIL}
//
// This file is part of {PROJECT_NAME} project.
// Repository: {file_url}
// GitHub: {GITHUB_PROFILE}
// Licensed under {LICENSE_TYPE}
//"""


def has_copyright_header(content: str) -> bool:
    return "Copyright (c)" in content and PROJECT_NAME in content


def has_disclaimer(content: str) -> bool:
    return "声明：本代码仅供学习和研究目的使用" in content


def find_insert_position_py(lines: List[str]) -> Tuple[int, bool]:
    insert_pos = 0
    has_encoding = False

    if lines and lines[0].startswith('#!'):
        insert_pos = 1

    for i in range(insert_pos, min(insert_pos + 2, len(lines))):
        if i < len(lines):
            line = lines[i].strip()
            if re.match(r'#.*coding[:=]\s*([-\w.]+)', line):
                has_encoding = True
                insert_pos = i + 1
                break

    return insert_pos, has_encoding


def find_insert_position_js(lines: List[str]) -> int:
    """JS files: insert at line 0 (or after 'use strict')"""
    if lines and lines[0].strip() in ('"use strict";', "'use strict';"):
        return 1
    return 0


def process_file(file_path: str, project_root: str, dry_run: bool = False) -> Tuple[bool, str]:
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
            lines = content.splitlines(keepends=True)

        if has_copyright_header(content):
            return False, f"  Already has header: {file_path}"

        file_type = detect_file_type(file_path)
        relative_path = get_file_relative_path(file_path, project_root)

        new_lines = []

        if file_type == 'python':
            insert_pos, has_encoding = find_insert_position_py(lines)
            if not has_encoding:
                new_lines.append("# -*- coding: utf-8 -*-\n")
            new_lines.extend(lines[:insert_pos])
            new_lines.append(generate_copyright_header_py(relative_path) + "\n")
            if not has_disclaimer(content):
                new_lines.append(DISCLAIMER_PY + "\n")
        elif file_type == 'javascript':
            insert_pos = find_insert_position_js(lines)
            new_lines.extend(lines[:insert_pos])
            new_lines.append(generate_copyright_header_js(relative_path) + "\n")
            if not has_disclaimer(content):
                new_lines.append(DISCLAIMER_JS + "\n")
        else:
            return False, f"  Skipped (unsupported type): {file_path}"

        # 添加空行分隔
        if insert_pos < len(lines) and lines[insert_pos].strip():
            new_lines.append("\n")

        new_lines.extend(lines[insert_pos:])

        if not dry_run:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.writelines(new_lines)
            return True, f"  Updated: {file_path}"
        else:
            return True, f"  Would update: {file_path}"

    except Exception as e:
        return False, f"  Error: {file_path}: {str(e)}"


def find_source_files(root_dir: str) -> List[str]:
    source_files = []
    for root, dirs, files in os.walk(root_dir):
        dirs[:] = [d for d in dirs if d not in EXCLUDE_DIRS and not d.startswith('.')]
        for file in files:
            if file.endswith(('.py', '.js')):
                source_files.append(os.path.join(root, file))
    return sorted(source_files)


def main():
    import argparse

    parser = argparse.ArgumentParser(description='File header copyright manager (Python + JS)')
    parser.add_argument('files', nargs='*', help='Files to process (default: all .py/.js files)')
    parser.add_argument('--dry-run', action='store_true', help='Check only, do not modify')
    parser.add_argument('--project-root', default=None, help='Project root directory')
    parser.add_argument('--check', action='store_true', help='Check mode, exit 1 if files need update')

    args = parser.parse_args()

    if args.project_root:
        project_root = os.path.abspath(args.project_root)
    else:
        project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

    print(f"Project root: {project_root}")
    print(f"Mode: {'CHECK' if args.check else 'DRY RUN' if args.dry_run else 'UPDATE'}")
    print("-" * 60)

    if args.files:
        files_to_process = [os.path.abspath(f) for f in args.files if f.endswith(('.py', '.js'))]
    else:
        files_to_process = find_source_files(project_root)

    print(f"Found {len(files_to_process)} source files\n")

    updated = skipped = errors = 0
    for file_path in files_to_process:
        modified, message = process_file(file_path, project_root, args.dry_run or args.check)
        print(message)
        if "Error" in message:
            errors += 1
        elif modified:
            updated += 1
        else:
            skipped += 1

    print(f"\n{'=' * 60}")
    print(f"Total: {len(files_to_process)} | Need update: {updated} | OK: {skipped} | Errors: {errors}")
    print(f"{'=' * 60}")

    if (args.check and updated > 0) or errors > 0:
        sys.exit(1)
    sys.exit(0)


if __name__ == '__main__':
    main()
