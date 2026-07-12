# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/services/video_transcoder.py
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
视频转码和切割服务

提供 720p 转码和按时间点切割视频的能力。
所有 ffmpeg 操作通过 asyncio.to_thread 包装，避免阻塞事件循环。
"""
import asyncio
import subprocess
from pathlib import Path
from typing import List, Dict

from utils.logger import logger


class VideoTranscodeError(Exception):
    """视频转码错误"""
    pass


class VideoTranscoder:
    """视频转码和切割服务"""

    async def get_duration(self, video_path: str) -> float:
        """
        获取视频时长（秒）

        Args:
            video_path: 视频文件路径

        Returns:
            时长（秒）

        Raises:
            VideoTranscodeError: 无法获取时长
        """
        def _probe():
            cmd = [
                "ffprobe", "-v", "error",
                "-show_entries", "format=duration",
                "-of", "default=noprint_wrappers=1:nokey=1",
                video_path
            ]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
            if result.returncode != 0:
                raise VideoTranscodeError(f"ffprobe failed: {result.stderr}")
            return float(result.stdout.strip())

        return await asyncio.to_thread(_probe)

    async def transcode_720p(self, input_path: str, output_path: str) -> str:
        """
        转码视频为 720p

        Args:
            input_path: 输入视频路径
            output_path: 输出视频路径

        Returns:
            输出文件路径

        Raises:
            VideoTranscodeError: 转码失败
        """
        def _transcode():
            Path(output_path).parent.mkdir(parents=True, exist_ok=True)
            cmd = [
                "ffmpeg", "-y", "-i", input_path,
                "-vf", "scale=-2:720",
                "-c:v", "libx264", "-preset", "fast", "-crf", "23",
                "-c:a", "aac", "-b:a", "128k",
                output_path
            ]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
            if result.returncode != 0:
                raise VideoTranscodeError(f"ffmpeg transcode failed: {result.stderr[-1500:]}")
            logger.info(f"[VideoTranscoder] Transcoded to 720p: {output_path}")
            return output_path

        return await asyncio.to_thread(_transcode)

    async def split_and_transcode(
        self,
        input_path: str,
        segments: List[Dict],
        output_dir: str,
    ) -> List[str]:
        """
        按时间点切割视频并转码 720p

        使用 -ss before -i 进行快速 seek，-t 指定输出时长。

        Args:
            input_path: 输入视频路径
            segments: 分段列表 [{"start": 0.0, "end": 185.5, "title": "..."}, ...]
            output_dir: 输出目录

        Returns:
            各段文件路径列表

        Raises:
            VideoTranscodeError: 切割/转码失败
        """
        if not Path(input_path).exists():
            raise VideoTranscodeError(f"Input video not found: {input_path}")

        Path(output_dir).mkdir(parents=True, exist_ok=True)
        output_paths = []

        for i, seg in enumerate(segments):
            output_path = str(Path(output_dir) / f"seg_{i + 1:03d}.mp4")

            def _split(inp=input_path, out=output_path, start=seg["start"], end=seg["end"], idx=i):
                duration = end - start
                # -ss before -i: 输入级快速 seek
                # -t: 输出时长（比 -to 更可靠）
                cmd = [
                    "ffmpeg", "-y",
                    "-ss", str(start),
                    "-i", inp,
                    "-t", str(duration),
                    "-vf", "scale=-2:720",
                    "-c:v", "libx264", "-preset", "fast", "-crf", "23",
                    "-c:a", "aac", "-b:a", "128k",
                    "-movflags", "+faststart",
                    out
                ]
                result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
                if result.returncode != 0:
                    # 截取 stderr 尾部（ffmpeg 实际错误在末尾，头部是版本信息）
                    stderr_tail = result.stderr[-1500:] if result.stderr else "(empty)"
                    raise VideoTranscodeError(
                        f"ffmpeg split failed for segment {idx + 1} "
                        f"(ss={start}, t={duration}): {stderr_tail}"
                    )
                return out

            path = await asyncio.to_thread(_split)
            output_paths.append(path)
            logger.info(f"[VideoTranscoder] Split segment {i + 1}/{len(segments)}: {seg['start']:.1f}s-{seg['end']:.1f}s -> {path}")

        return output_paths
