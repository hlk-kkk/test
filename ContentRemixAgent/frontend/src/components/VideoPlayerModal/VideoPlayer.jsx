/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/VideoPlayerModal/VideoPlayer.jsx
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

import React, { useState, useRef } from 'react';

/**
 * VideoPlayer - 原生 HTML5 视频播放器
 * 支持自动检测视频方向（横版/竖版）并自适应布局
 */
function VideoPlayer({ url }) {
  const [isPortrait, setIsPortrait] = useState(false);
  const videoRef = useRef(null);

  if (!url) {
    return <div className="video-player-error">No video URL</div>;
  }

  // 检测视频方向
  const handleLoadedMetadata = (e) => {
    const video = e.target;
    const aspectRatio = video.videoWidth / video.videoHeight;
    // 如果宽高比 < 1，说明是竖版视频
    setIsPortrait(aspectRatio < 1);
  };

  return (
    <div
      style={{
        width: '100%',
        height: '100%',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        background: '#000',
      }}
    >
      <video
        ref={videoRef}
        src={url}
        controls
        autoPlay
        onLoadedMetadata={handleLoadedMetadata}
        style={{
          maxWidth: isPortrait ? '400px' : '100%',
          maxHeight: '100%',
          width: isPortrait ? 'auto' : '100%',
          height: isPortrait ? '100%' : 'auto',
          objectFit: 'contain',
        }}
      />
    </div>
  );
}

export default VideoPlayer;
