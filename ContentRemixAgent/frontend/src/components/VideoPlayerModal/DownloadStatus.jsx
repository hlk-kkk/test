/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/VideoPlayerModal/DownloadStatus.jsx
 * GitHub: https://github.com/NanmiCoder
 * Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
 *
 * 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
 * 1. 不得用于任何商业用途。
 * 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
 * 3. 不得进行大规模爬取或对平台造成运营干扰。
 * 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
 * 5. 不得用于任何非法或不当的用途。
 *
 * 详细许可条款请参阅项目根目录下的LICENSE文件。
 * 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。
 */

import React from 'react';
import { CheckCircle, AlertCircle, FolderOpen, ExternalLink } from 'lucide-react';
import { useTranslation } from 'react-i18next';

/**
 * DownloadStatus - 下载状态显示组件
 *
 * 显示视频下载完成状态或未下载提示
 */
function DownloadStatus({ localVideoUrl, originalVideoUrl, fileSize }) {
  const { t } = useTranslation();

  // 从 URL 中提取文件路径
  const getDisplayPath = (url) => {
    if (!url) return null;
    // 处理本地文件路径，截取后半部分显示
    const parts = url.split('/');
    if (parts.length > 3) {
      return '.../' + parts.slice(-3).join('/');
    }
    return url;
  };

  // 格式化文件大小
  const formatFileSize = (bytes) => {
    if (!bytes) return null;
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  if (localVideoUrl) {
    return (
      <div className="download-status download-status-success">
        <div className="download-status-icon">
          <CheckCircle size={16} className="text-accent-green" />
        </div>
        <div className="download-status-content">
          <span className="download-status-label">{t('videoPlayer.downloaded')}</span>
          {fileSize && (
            <span className="download-status-size">{formatFileSize(fileSize)}</span>
          )}
          <span className="download-status-path" title={localVideoUrl}>
            <FolderOpen size={12} />
            {getDisplayPath(localVideoUrl)}
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className="download-status download-status-pending">
      <div className="download-status-icon">
        <AlertCircle size={16} className="text-accent-orange" />
      </div>
      <div className="download-status-content">
        <span className="download-status-label">{t('videoPlayer.notDownloaded')}</span>
        {originalVideoUrl && (
          <a
            href={originalVideoUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="download-status-link"
          >
            <ExternalLink size={12} />
            {t('videoPlayer.openOriginal')}
          </a>
        )}
      </div>
    </div>
  );
}

export default DownloadStatus;
