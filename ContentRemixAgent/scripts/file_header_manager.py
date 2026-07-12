#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/scripts/file_header_manager.py
# GitHub: https://github.com/NanmiCoder
# Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
#
# 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
# 1. 不得用于任何商业用途。
# 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
# 3. 不得进行大规模爬取或对平台造成运营干扰。
# 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
# 5. 不得用于任何非法或不当的用途。
#
# 详细许可条款请参阅项目根目录下的LICENSE文件。
# 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。

"""
文件头版权声明管理工具

功能：
- 自动为 Python 和 JS/JSX 文件添加标准化的版权声明和免责声明
- 智能检测现有文件头（编码声明、shebang、ESLint 注释等）
- 在合适位置插入版权信息，不破坏现有内容
- 支持批量处理和单文件检查模式
"""

import os
import re
import sys
from enum import Enum
from typing import List, Tuple

# 项目配置
REPO_URL = "https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent"
GITHUB_PROFILE = "https://github.com/NanmiCoder"
EMAIL = "relakkes@gmail.com"
COPYRIGHT_YEAR = "2026"
LICENSE_TYPE = "NON-COMMERCIAL LEARNING LICENSE 1.1"
PROJECT_NAME = "MediaCrawlerPro-ContentRemixAgent"

# 排除的目录
EXCLUDE_DIRS = [
    'venv', '.venv', 'node_modules', '__pycache__', '.git',
    'build', 'dist', '.eggs', '.pytest_cache', '.mypy_cache',
    '.ruff_cache', 'htmlcov', '.tox', 'egg-info'
]


class FileType(Enum):
    PYTHON = "python"
    JAVASCRIPT = "javascript"
    UNKNOWN = "unknown"


# Python 免责声明
PYTHON_DISCLAIMER = """# 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
# 1. 不得用于任何商业用途。
# 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
# 3. 不得进行大规模爬取或对平台造成运营干扰。
# 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
# 5. 不得用于任何非法或不当的用途。
#
# 详细许可条款请参阅项目根目录下的LICENSE文件。
# 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。"""

# JS/JSX 免责声明
JS_DISCLAIMER = """ * 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
 * 1. 不得用于任何商业用途。
 * 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
 * 3. 不得进行大规模爬取或对平台造成运营干扰。
 * 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
 * 5. 不得用于任何非法或不当的用途。
 *
 * 详细许可条款请参阅项目根目录下的LICENSE文件。
 * 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。"""


def detect_file_type(file_path: str) -> FileType:
    """检测文件类型"""
    ext = os.path.splitext(file_path)[1].lower()
    if ext == '.py':
        return FileType.PYTHON
    elif ext in ['.js', '.jsx', '.ts', '.tsx', '.mjs', '.cjs']:
        return FileType.JAVASCRIPT
    return FileType.UNKNOWN


def get_file_relative_path(file_path: str, project_root: str) -> str:
    """获取文件相对于项目根目录的路径"""
    return os.path.relpath(file_path, project_root)


def generate_python_header(relative_path: str) -> str:
    """生成 Python 文件的版权声明头部"""
    file_url = f"{REPO_URL}/blob/main/{relative_path}"

    header = f"""# Copyright (c) {COPYRIGHT_YEAR} {EMAIL}
#
# This file is part of {PROJECT_NAME} project.
# Repository: {file_url}
# GitHub: {GITHUB_PROFILE}
# Licensed under {LICENSE_TYPE}
#"""

    return header


def generate_js_header(relative_path: str) -> str:
    """生成 JS/JSX 文件的版权声明头部"""
    file_url = f"{REPO_URL}/blob/main/{relative_path}"

    header = f"""/**
 * Copyright (c) {COPYRIGHT_YEAR} {EMAIL}
 *
 * This file is part of {PROJECT_NAME} project.
 * Repository: {file_url}
 * GitHub: {GITHUB_PROFILE}
 * Licensed under {LICENSE_TYPE}
 *
{JS_DISCLAIMER}
 */"""

    return header


def has_copyright_header(content: str, file_type: FileType) -> bool:
    """检查文件是否已包含版权声明（在文件头部注释中）"""
    lines = content.split('\n')

    if file_type == FileType.PYTHON:
        # 检查前 20 行是否有标准的 Python 版权声明注释
        for line in lines[:20]:
            if line.startswith('# Copyright (c)') and EMAIL in line:
                return True
        return False

    elif file_type == FileType.JAVASCRIPT:
        # 检查前 30 行是否有标准的 JS 版权声明注释
        header_text = '\n'.join(lines[:30])
        if '* Copyright (c)' in header_text and EMAIL in header_text:
            return True
        return False

    return False


def has_disclaimer(content: str) -> bool:
    """检查文件是否已包含免责声明"""
    return "声明：本代码仅供学习和研究目的使用" in content


def find_python_insert_position(lines: List[str]) -> Tuple[int, bool]:
    """
    找到 Python 文件插入版权声明的位置

    Returns:
        (插入行号, 是否需要在前面添加编码声明)
    """
    insert_pos = 0
    has_encoding = False

    # 检查第一行是否是 shebang
    if lines and lines[0].startswith('#!'):
        insert_pos = 1

    # 检查编码声明（通常在第1或2行）
    for i in range(insert_pos, min(insert_pos + 2, len(lines))):
        if i < len(lines):
            line = lines[i].strip()
            # 匹配 # -*- coding: utf-8 -*- 或 # coding: utf-8 等格式
            if re.match(r'#.*coding[:=]\s*([-\w.]+)', line):
                has_encoding = True
                insert_pos = i + 1
                break

    return insert_pos, has_encoding


def find_js_insert_position(lines: List[str]) -> int:
    """
    找到 JS/JSX 文件插入版权声明的位置

    Returns:
        插入行号
    """
    insert_pos = 0

    # 检查开头是否有需要保留的内容
    for i, line in enumerate(lines):
        stripped = line.strip()

        # 跳过空行
        if not stripped:
            insert_pos = i + 1
            continue

        # 保留 'use strict' 声明
        if stripped in ["'use strict';", '"use strict";']:
            insert_pos = i + 1
            continue

        # 保留 ESLint 禁用注释
        if stripped.startswith('/* eslint-disable') or stripped.startswith('// eslint-disable'):
            insert_pos = i + 1
            continue

        # 遇到其他内容，停止
        break

    return insert_pos


def process_python_file(file_path: str, project_root: str, dry_run: bool = False) -> Tuple[bool, str]:
    """处理单个 Python 文件"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
            lines = content.splitlines(keepends=True)

        # 如果已经有版权声明，跳过
        if has_copyright_header(content, FileType.PYTHON):
            return False, f"✓ Already has copyright header: {file_path}"

        # 获取相对路径
        relative_path = get_file_relative_path(file_path, project_root)

        # 生成版权声明
        copyright_header = generate_python_header(relative_path)

        # 查找插入位置
        insert_pos, has_encoding = find_python_insert_position(lines)

        # 构建新的文件内容
        new_lines = []

        # 如果没有编码声明，添加一个
        if not has_encoding:
            new_lines.append("# -*- coding: utf-8 -*-\n")

        # 添加前面的部分（shebang 和编码声明）
        new_lines.extend(lines[:insert_pos])

        # 添加版权声明
        new_lines.append(copyright_header + "\n")

        # 如果文件没有免责声明，添加免责声明
        if not has_disclaimer(content):
            new_lines.append(PYTHON_DISCLAIMER + "\n")

        # 添加一个空行（如果下一行不是空行）
        if insert_pos < len(lines) and lines[insert_pos].strip():
            new_lines.append("\n")

        # 添加剩余的内容
        new_lines.extend(lines[insert_pos:])

        # 如果不是 dry run，写入文件
        if not dry_run:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.writelines(new_lines)
            return True, f"✓ Updated: {file_path}"
        else:
            return True, f"→ Would update: {file_path}"

    except Exception as e:
        return False, f"✗ Error processing {file_path}: {str(e)}"


def process_js_file(file_path: str, project_root: str, dry_run: bool = False) -> Tuple[bool, str]:
    """处理单个 JS/JSX 文件"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
            lines = content.splitlines(keepends=True)

        # 如果已经有版权声明，跳过
        if has_copyright_header(content, FileType.JAVASCRIPT):
            return False, f"✓ Already has copyright header: {file_path}"

        # 获取相对路径
        relative_path = get_file_relative_path(file_path, project_root)

        # 生成版权声明（包含免责声明）
        copyright_header = generate_js_header(relative_path)

        # 查找插入位置
        insert_pos = find_js_insert_position(lines)

        # 构建新的文件内容
        new_lines = []

        # 添加前面的部分
        new_lines.extend(lines[:insert_pos])

        # 添加版权声明
        new_lines.append(copyright_header + "\n")

        # 添加一个空行（如果下一行不是空行）
        if insert_pos < len(lines) and lines[insert_pos].strip():
            new_lines.append("\n")

        # 添加剩余的内容
        new_lines.extend(lines[insert_pos:])

        # 如果不是 dry run，写入文件
        if not dry_run:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.writelines(new_lines)
            return True, f"✓ Updated: {file_path}"
        else:
            return True, f"→ Would update: {file_path}"

    except Exception as e:
        return False, f"✗ Error processing {file_path}: {str(e)}"


def process_file(file_path: str, project_root: str, dry_run: bool = False) -> Tuple[bool, str]:
    """处理单个文件（自动检测类型）"""
    file_type = detect_file_type(file_path)

    if file_type == FileType.PYTHON:
        return process_python_file(file_path, project_root, dry_run)
    elif file_type == FileType.JAVASCRIPT:
        return process_js_file(file_path, project_root, dry_run)
    else:
        return False, f"⊘ Skipped (unsupported type): {file_path}"


def find_source_files(root_dir: str, extensions: List[str] = None) -> List[str]:
    """
    查找所有源代码文件

    Args:
        root_dir: 根目录
        extensions: 文件扩展名列表

    Returns:
        源代码文件路径列表
    """
    if extensions is None:
        extensions = ['.py', '.js', '.jsx', '.ts', '.tsx']

    source_files = []

    for root, dirs, files in os.walk(root_dir):
        # 排除特定目录
        dirs[:] = [d for d in dirs if d not in EXCLUDE_DIRS and not d.startswith('.')]

        for file in files:
            if any(file.endswith(ext) for ext in extensions):
                source_files.append(os.path.join(root, file))

    return sorted(source_files)


def main():
    """主函数"""
    import argparse

    parser = argparse.ArgumentParser(description='文件头版权声明管理工具（支持 Python 和 JS/JSX）')
    parser.add_argument('files', nargs='*', help='要处理的文件路径（可选，默认处理所有源代码文件）')
    parser.add_argument('--dry-run', action='store_true', help='仅检查不修改文件')
    parser.add_argument('--project-root', default=None, help='项目根目录（默认为当前目录）')
    parser.add_argument('--check', action='store_true', help='检查模式，如果有文件缺少版权声明则返回非零退出码')
    parser.add_argument('--python-only', action='store_true', help='仅处理 Python 文件')
    parser.add_argument('--js-only', action='store_true', help='仅处理 JS/JSX 文件')

    args = parser.parse_args()

    # 确定项目根目录
    if args.project_root:
        project_root = os.path.abspath(args.project_root)
    else:
        # 假设此脚本在 scripts/ 目录下
        project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

    print(f"Project root: {project_root}")
    print(f"Mode: {'CHECK' if args.check else 'DRY RUN' if args.dry_run else 'UPDATE'}")
    print("-" * 60)

    # 确定文件扩展名
    extensions = None
    if args.python_only:
        extensions = ['.py']
    elif args.js_only:
        extensions = ['.js', '.jsx', '.ts', '.tsx']

    # 获取要处理的文件列表
    if args.files:
        # 处理指定的文件
        files_to_process = [os.path.abspath(f) for f in args.files]
        # 过滤支持的文件类型
        files_to_process = [f for f in files_to_process if detect_file_type(f) != FileType.UNKNOWN]
    else:
        # 处理所有源代码文件
        files_to_process = find_source_files(project_root, extensions)

    print(f"Found {len(files_to_process)} source files to process\n")

    # 处理文件
    updated_count = 0
    skipped_count = 0
    error_count = 0

    for file_path in files_to_process:
        modified, message = process_file(file_path, project_root, args.dry_run or args.check)
        print(message)

        if "Error" in message:
            error_count += 1
        elif modified:
            updated_count += 1
        else:
            skipped_count += 1

    # 打印汇总
    print("\n" + "=" * 60)
    print("Summary:")
    print(f"  Total files: {len(files_to_process)}")
    print(f"  Updated/Need update: {updated_count}")
    print(f"  Already compliant: {skipped_count}")
    print(f"  Errors: {error_count}")
    print("=" * 60)

    # 如果是 check 模式且有文件需要更新，返回非零退出码
    if args.check and updated_count > 0:
        print(f"\n⚠ {updated_count} file(s) missing copyright headers!")
        sys.exit(1)
    elif error_count > 0:
        sys.exit(1)
    else:
        sys.exit(0)


if __name__ == '__main__':
    main()
