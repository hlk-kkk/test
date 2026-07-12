/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/VideoPlayerModal/index.jsx
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
import { ExternalLink, Video } from 'lucide-react';
import Modal from '../common/Modal';
import VideoPlayer from './VideoPlayer';
import DownloadStatus from './DownloadStatus';

/**
 * VideoPlayerModal - 视频播放对话框
 *
 * 组合 Modal + VideoPlayer + DownloadStatus
 * 支持三种状态：可播放、未下载、加载中
 */
function VideoPlayerModal({
  isOpen,
  onClose,
  title,
  localVideoUrl,
  originalVideoUrl,
  coverUrl,
  fileSize,
}) {
  // 外部链接按钮
  const headerActions = originalVideoUrl ? (
    <a
      href={originalVideoUrl}
      target="_blank"
      rel="noopener noreferrer"
      className="video-modal-external-link"
    >
      <ExternalLink size={16} />
      <span>Original</span>
    </a>
  ) : null;

  // 判断是否可以播放本地视频
  const canPlay = Boolean(localVideoUrl);

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title || 'Video Player'}
      headerActions={headerActions}
      maxWidth="4xl"
    >
      <div className="video-modal-body">
        {/* Video Player Area */}
        <div className="video-modal-player-wrapper">
          {canPlay ? (
            <VideoPlayer url={localVideoUrl} coverUrl={coverUrl} />
          ) : (
            <div className="video-modal-no-video">
              <div className="video-modal-no-video-icon">
                <Video size={48} />
              </div>
              <p className="video-modal-no-video-text">
                Video not downloaded yet
              </p>
              {originalVideoUrl && (
                <a
                  href={originalVideoUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="video-modal-no-video-link"
                >
                  <ExternalLink size={16} />
                  Watch on original platform
                </a>
              )}
            </div>
          )}
        </div>

        {/* Download Status */}
        <div className="video-modal-status">
          <DownloadStatus
            localVideoUrl={localVideoUrl}
            originalVideoUrl={originalVideoUrl}
            fileSize={fileSize}
          />
        </div>
      </div>
    </Modal>
  );
}

export default VideoPlayerModal;
