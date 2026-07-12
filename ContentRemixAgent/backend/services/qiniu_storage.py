# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/services/qiniu_storage.py
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
七牛云存储服务 - 上传音频/视频文件并获取公网 URL

用于全模态模型（omni）的媒体直传场景。
文件上传后自动设置过期删除（默认 1 天）。
"""
import asyncio
from typing import Optional

from config import settings
from utils.logger import logger


class QiniuStorageError(Exception):
    """七牛云存储错误"""
    pass


class QiniuStorageService:
    """
    七牛云存储服务

    上传本地音频文件到七牛云，返回公网 URL。
    文件通过 deleteAfterDays 策略自动过期删除。
    """

    def is_available(self) -> bool:
        """检查七牛配置是否完整"""
        return all([
            settings.QINIU_ACCESS_KEY,
            settings.QINIU_SECRET_KEY,
            settings.QINIU_BUCKET,
            settings.QINIU_DOMAIN,
        ])

    async def upload_audio(self, local_path: str, key: str) -> str:
        """
        上传音频文件到七牛云

        Args:
            local_path: 本地音频文件路径
            key: 七牛存储 key，如 audio/{session_id}/{content_id}.wav

        Returns:
            公网访问 URL

        Raises:
            QiniuStorageError: 上传失败或配置不完整
        """
        if not self.is_available():
            raise QiniuStorageError(
                "七牛云配置不完整，请在 .env 中配置 QINIU_ACCESS_KEY, QINIU_SECRET_KEY, QINIU_BUCKET, QINIU_DOMAIN"
            )

        try:
            url = await asyncio.to_thread(self._sync_upload, local_path, key)
            logger.info(f"[Qiniu] Audio uploaded: {key} -> {url}")
            return url
        except QiniuStorageError:
            raise
        except Exception as e:
            raise QiniuStorageError(f"上传失败: {e}") from e

    def _sync_upload(self, local_path: str, key: str) -> str:
        """同步上传（在线程池中执行）"""
        import qiniu

        auth = qiniu.Auth(settings.QINIU_ACCESS_KEY, settings.QINIU_SECRET_KEY)

        # 上传策略：文件 N 天后自动删除
        policy = {
            "deleteAfterDays": settings.QINIU_DELETE_AFTER_DAYS,
        }

        token = auth.upload_token(
            settings.QINIU_BUCKET, key, 3600, policy=policy
        )

        ret, info = qiniu.put_file(token, key, local_path, mime_type="audio/wav")

        if ret is None:
            raise QiniuStorageError(f"七牛上传失败: {info}")

        # 拼接公网 URL
        domain = settings.QINIU_DOMAIN.rstrip("/")
        if not domain.startswith(("http://", "https://")):
            domain = f"http://{domain}"

        return f"{domain}/{key}"

    async def upload_video(self, local_path: str, key: str) -> str:
        """
        上传视频文件到七牛云

        Args:
            local_path: 本地视频文件路径
            key: 七牛存储 key，如 video/{session_id}/{content_id}/seg_001.mp4

        Returns:
            公网访问 URL

        Raises:
            QiniuStorageError: 上传失败或配置不完整
        """
        if not self.is_available():
            raise QiniuStorageError(
                "七牛云配置不完整，请在 .env 中配置 QINIU_ACCESS_KEY, QINIU_SECRET_KEY, QINIU_BUCKET, QINIU_DOMAIN"
            )

        try:
            url = await asyncio.to_thread(self._sync_upload_video, local_path, key)
            logger.info(f"[Qiniu] Video uploaded: {key} -> {url}")
            return url
        except QiniuStorageError:
            raise
        except Exception as e:
            raise QiniuStorageError(f"视频上传失败: {e}") from e

    def _sync_upload_video(self, local_path: str, key: str) -> str:
        """同步上传视频（在线程池中执行）"""
        import qiniu

        auth = qiniu.Auth(settings.QINIU_ACCESS_KEY, settings.QINIU_SECRET_KEY)

        policy = {
            "deleteAfterDays": settings.QINIU_DELETE_AFTER_DAYS,
        }

        token = auth.upload_token(
            settings.QINIU_BUCKET, key, 3600, policy=policy
        )

        ret, info = qiniu.put_file(token, key, local_path, mime_type="video/mp4")

        if ret is None:
            raise QiniuStorageError(f"七牛视频上传失败: {info}")

        domain = settings.QINIU_DOMAIN.rstrip("/")
        if not domain.startswith(("http://", "https://")):
            domain = f"http://{domain}"

        return f"{domain}/{key}"
