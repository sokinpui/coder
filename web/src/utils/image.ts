export const fileToDataUrl = (file: File): Promise<string> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = (error) => reject(error);
    reader.readAsDataURL(file);
  });
};

export const processImage = (dataUrl: string, quality = 0.9): Promise<string> => {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => {
      const canvas = document.createElement('canvas');
      canvas.width = img.width;
      canvas.height = img.height;
      const ctx = canvas.getContext('2d');
      if (!ctx) {
        return reject(new Error('Failed to get 2D context for canvas.'));
      }
      ctx.drawImage(img, 0, 0);
      const processedDataUrl = canvas.toDataURL('image/jpeg', quality);
      resolve(processedDataUrl);
    };
    img.onerror = () => reject(new Error('Failed to load image into canvas.'));
    img.src = dataUrl;
  });
};
