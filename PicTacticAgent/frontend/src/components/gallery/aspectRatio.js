const RATIO_CSS = {
  "1:1": "1 / 1",
  "16:9": "16 / 9",
  "9:16": "9 / 16",
  "4:3": "4 / 3",
  "3:4": "3 / 4",
};

const imageAspectRatioCache = new Map();
const pendingAspectRatioLoads = new Map();

export function getAspectRatioCss(aspectRatio) {
  return RATIO_CSS[aspectRatio] || "16 / 9";
}

function toPositiveNumber(value) {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

function getRatioFromDimensions(width, height) {
  const validWidth = toPositiveNumber(width);
  const validHeight = toPositiveNumber(height);
  if (!validWidth || !validHeight) return null;
  return `${validWidth} / ${validHeight}`;
}

function getRatioFromImageData(image) {
  const metadata = image?.metadata || {};

  const dimensionPairs = [
    [image?.width, image?.height],
    [image?.pixel_width, image?.pixel_height],
    [metadata.width, metadata.height],
    [metadata.pixel_width, metadata.pixel_height],
    [metadata.image_width, metadata.image_height],
    [metadata.dimensions?.width, metadata.dimensions?.height],
  ];

  for (const [width, height] of dimensionPairs) {
    const ratio = getRatioFromDimensions(width, height);
    if (ratio) return ratio;
  }

  return null;
}

export function getCachedImageAspectRatio(url) {
  if (!url) return null;
  return imageAspectRatioCache.get(url) || null;
}

export function cacheImageAspectRatio(url, width, height) {
  const ratio = getRatioFromDimensions(width, height);
  if (!url || !ratio) return null;
  imageAspectRatioCache.set(url, ratio);
  return ratio;
}

export function hasReliableImageAspectRatio(image) {
  return Boolean(
    getRatioFromImageData(image) ||
    getCachedImageAspectRatio(image?.thumbnail_url) ||
    getCachedImageAspectRatio(image?.url)
  );
}

export function preloadImageAspectRatio(url) {
  if (!url) return Promise.resolve(null);

  const cached = getCachedImageAspectRatio(url);
  if (cached) return Promise.resolve(cached);

  const pending = pendingAspectRatioLoads.get(url);
  if (pending) return pending;

  const loadPromise = new Promise((resolve) => {
    const img = new Image();
    let done = false;

    const finalize = (ratio) => {
      if (done) return;
      done = true;
      pendingAspectRatioLoads.delete(url);
      resolve(ratio);
    };

    img.onload = () => {
      finalize(cacheImageAspectRatio(url, img.naturalWidth, img.naturalHeight));
    };
    img.onerror = () => finalize(null);
    img.src = url;
  });

  pendingAspectRatioLoads.set(url, loadPromise);
  return loadPromise;
}

export function getImageAspectRatio(image, fallbackAspectRatio = "16:9") {
  return (
    getRatioFromImageData(image) ||
    getCachedImageAspectRatio(image?.thumbnail_url) ||
    getCachedImageAspectRatio(image?.url) ||
    getAspectRatioCss(fallbackAspectRatio)
  );
}
