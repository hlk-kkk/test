/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/ImageGalleryModal/index.jsx
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

import React, { useState } from 'react';
import { ExternalLink, Image as ImageIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import Modal from '../common/Modal';
import ImageCarousel from './ImageCarousel';

/**
 * ImageGalleryModal - 图片轮播对话框
 *
 * 支持:
 * - 图片轮播展示
 * - 图片计数器
 * - mixed 类型时显示"观看视频"链接
 */
function ImageGalleryModal({
  isOpen,
  onClose,
  images,
  title,
  originalUrl,
  videoUrl,
}) {
  const { t } = useTranslation();
  const [currentIndex, setCurrentIndex] = useState(0);

  const totalImages = images?.length || 0;

  // 重置 index 当 modal 关闭时
  const handleClose = () => {
    setCurrentIndex(0);
    onClose();
  };

  // 外部链接按钮
  const headerActions = (
    <div className="flex items-center gap-2">
      {/* 图片计数器 */}
      {totalImages > 0 && (
        <span className="image-modal-counter">
          {currentIndex + 1} / {totalImages}
        </span>
      )}
      {/* 原始链接 */}
      {originalUrl && (
        <a
          href={originalUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="video-modal-external-link"
        >
          <ExternalLink size={16} />
          <span>Original</span>
        </a>
      )}
    </div>
  );

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title={title || t('imageGallery.title', 'Image Gallery')}
      headerActions={headerActions}
      maxWidth="4xl"
    >
      <div className="image-modal-body">
        {/* 图片轮播区域 */}
        <div className="image-modal-carousel-wrapper">
          {totalImages > 0 ? (
            <ImageCarousel
              images={images}
              currentIndex={currentIndex}
              onIndexChange={setCurrentIndex}
            />
          ) : (
            <div className="image-modal-no-images">
              <div className="image-modal-no-images-icon">
                <ImageIcon size={48} />
              </div>
              <p className="image-modal-no-images-text">
                {t('imageGallery.noImages', 'No images available')}
              </p>
            </div>
          )}
        </div>

        {/* Mixed 类型时显示视频链接 */}
        {videoUrl && (
          <div className="image-modal-video-link">
            <a
              href={videoUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="image-modal-watch-video"
            >
              <span className="material-symbols-outlined text-lg">
                play_circle
              </span>
              <span>{t('imageGallery.watchVideo', 'Watch Video')}</span>
              <ExternalLink size={14} />
            </a>
          </div>
        )}
      </div>
    </Modal>
  );
}

export default ImageGalleryModal;
