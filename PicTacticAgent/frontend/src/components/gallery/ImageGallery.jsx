import { useEffect, useMemo, useState } from "react";
import { ImageCard } from "./ImageCard";
import { ImageEditDialog } from "./ImageEditDialog";
import { ImageModal } from "./ImageModal";
import { ImageSkeleton } from "./ImageSkeleton";
import { VideoCard } from "./VideoCard";
import { VideoModal } from "./VideoModal";
import { MasonryGrid } from "./MasonryGrid";
import { hasReliableImageAspectRatio, preloadImageAspectRatio } from "./aspectRatio";
import { useLocale } from "@/hooks/useLocale";

export function ImageGallery({
  images,
  videos,
  title = "Results",
  aspectRatio = "16:9",
  imageEdit,
  enhancedPrompt = "",
  mediaType = "image",
}) {
  const [selectedIndex, setSelectedIndex] = useState(null);
  const [localImages, setLocalImages] = useState(images || []);
  const [editDialogState, setEditDialogState] = useState(null);
  const [selectedVideo, setSelectedVideo] = useState(null);
  const { t } = useLocale();

  const isVideo = mediaType === "video";
  const items = isVideo ? (videos || []) : (images || []);

  // Sync local images when prop images change
  useEffect(() => {
    if (!isVideo) {
      setLocalImages(images || []);
    }
  }, [images, isVideo]);

  const sortedImages = useMemo(
    () => isVideo ? [] : [...localImages].sort((a, b) => b.score - a.score),
    [localImages, isVideo]
  );

  const sortedVideos = useMemo(
    () => isVideo ? [...(videos || [])].sort((a, b) => (a.round_number || 0) - (b.round_number || 0)) : [],
    [videos, isVideo]
  );

  const unknownUrls = useMemo(
    () => isVideo ? [] : Array.from(
      new Set(
        sortedImages
          .filter((image) => !hasReliableImageAspectRatio(image))
          .map((image) => image.thumbnail_url || image.url)
          .filter(Boolean)
      )
    ),
    [sortedImages, isVideo]
  );
  const preloadKey = useMemo(() => unknownUrls.join("|"), [unknownUrls]);
  const [completedPreloadKey, setCompletedPreloadKey] = useState("");
  const isLayoutReady = isVideo || !preloadKey || completedPreloadKey === preloadKey;

  useEffect(() => {
    if (!preloadKey || isVideo) return undefined;

    let cancelled = false;

    Promise.all(unknownUrls.map((url) => preloadImageAspectRatio(url)))
      .finally(() => {
        if (!cancelled) {
          setCompletedPreloadKey(preloadKey);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [preloadKey, unknownUrls, isVideo]);

  const openEditDialog = (image) => {
    if (!imageEdit) return;

    const imageIndex = localImages.findIndex((img) => img.id === image.id);
    if (imageIndex < 0) return;

    imageEdit.startEdit(image.id);
    setEditDialogState({
      index: imageIndex,
      history: [localImages[imageIndex]],
    });
  };

  const closeEditDialog = () => {
    imageEdit?.cancelEdit();
    setEditDialogState(null);
  };

  const editingImage = useMemo(() => {
    if (!editDialogState) return null;
    return localImages[editDialogState.index] || null;
  }, [editDialogState, localImages]);

  const handleEditConfirm = async (editPrompt, enhance) => {
    if (!imageEdit || !editDialogState || !editingImage) return;

    const targetIndex = editDialogState.index;
    const previousImage = editingImage;

    try {
      await imageEdit.confirmEdit(previousImage.id, editPrompt, enhance, (_id, newImage) => {
        const mergedImage = {
          ...previousImage,
          ...newImage,
          score: newImage.score ?? previousImage.score ?? 0,
          score_details: newImage.score_details ?? previousImage.score_details,
        };

        setLocalImages((prev) => {
          if (targetIndex < 0 || targetIndex >= prev.length) return prev;
          const next = [...prev];
          next[targetIndex] = mergedImage;
          return next;
        });

        setEditDialogState((prev) => {
          if (!prev || prev.index !== targetIndex) return prev;
          return {
            ...prev,
            history: [...prev.history, mergedImage],
          };
        });
      });
    } catch (error) {
      console.error("Edit failed:", error);
    }
  };

  const handleUndo = () => {
    if (!imageEdit || !editDialogState || editDialogState.history.length <= 1) return;

    const { index, history } = editDialogState;
    const currentImage = history[history.length - 1];
    const restoredImage = history[history.length - 2];

    setLocalImages((prev) => {
      if (index < 0 || index >= prev.length) return prev;
      const next = [...prev];
      next[index] = restoredImage;
      return next;
    });

    imageEdit.undoEdit(currentImage.id);
    setEditDialogState((prev) => {
      if (!prev || prev.index !== index) return prev;
      return {
        ...prev,
        history: prev.history.slice(0, -1),
      };
    });
  };

  if (items.length === 0) return null;

  // Video gallery
  if (isVideo) {
    return (
      <div className="w-full animate-slide-up">
        <div className="mb-4 flex items-center gap-3">
          <h2 className="text-sm font-semibold uppercase tracking-[0.08em] text-foreground font-display">{title}</h2>
          <div className="h-px flex-1 bg-border/30" />
          <span className="text-[10px] tabular-nums text-muted-foreground">
            {t("video.totalVideos", { count: sortedVideos.length })}
          </span>
        </div>
        <MasonryGrid
          className="stagger-children columns-1 gap-3 sm:columns-2 lg:columns-3"
          itemClassName="mb-3"
        >
          {sortedVideos.map((video, index) => (
            <div key={video.id}>
              <VideoCard
                video={video}
                rank={index}
                aspectRatio={aspectRatio}
                onClick={(v) => setSelectedVideo(v)}
              />
            </div>
          ))}
        </MasonryGrid>

        {selectedVideo && (
          <VideoModal
            video={selectedVideo}
            isOpen={true}
            onClose={() => setSelectedVideo(null)}
          />
        )}
      </div>
    );
  }

  // Image gallery (original behavior)
  return (
    <div className="w-full animate-slide-up">
      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-sm font-semibold uppercase tracking-[0.08em] text-foreground font-display">{title}</h2>
        <div className="h-px flex-1 bg-border/30" />
        <span className="text-[10px] tabular-nums text-muted-foreground">{t("gallery.images", { count: items.length })}</span>
      </div>
      <MasonryGrid
        className="stagger-children columns-1 gap-3 sm:columns-2 lg:columns-3"
        itemClassName="mb-3"
      >
        {!isLayoutReady &&
          sortedImages.map((image) => (
            <ImageSkeleton key={`loading-${image.id}`} aspectRatio={aspectRatio} />
          ))}

        {isLayoutReady &&
          sortedImages.map((image, index) => (
            <div key={image.id}>
              <ImageCard
                image={image}
                rank={index}
                onClick={() => setSelectedIndex(index)}
                aspectRatio={aspectRatio}
                onEdit={imageEdit ? () => openEditDialog(image) : undefined}
                isEditLoading={imageEdit?.isLoading(image.id)}
              />
            </div>
          ))}
      </MasonryGrid>

      {selectedIndex != null && (
        <ImageModal
          images={sortedImages}
          currentIndex={selectedIndex}
          onNavigate={setSelectedIndex}
          onClose={() => setSelectedIndex(null)}
          onEdit={imageEdit ? (img) => {
            setSelectedIndex(null);
            openEditDialog(img);
          } : undefined}
        />
      )}

      {editDialogState && editingImage && (
        <ImageEditDialog
          key={editingImage.id}
          image={editingImage}
          defaultPrompt={enhancedPrompt}
          isLoading={imageEdit?.isLoading(editingImage.id)}
          error={imageEdit?.error}
          editCount={editDialogState.history.length - 1}
          canUndo={editDialogState.history.length > 1}
          onConfirm={handleEditConfirm}
          onUndo={handleUndo}
          onClose={closeEditDialog}
        />
      )}
    </div>
  );
}
