import { Upload } from 'tus-js-client';

class TusMultiFileUploader {
  constructor(endpoint = 'http://localhost:1080/files/') {
    this.endpoint = endpoint;
    this.uploads = new Map();
    this.onProgress = null;
    this.onComplete = null;
    this.onError = null;
  }

  /**
   * 并行上传多个文件
   * @param {FileList|File[]} files - 要上传的文件列表
   * @param {Object} options - 上传配置选项
   */
  async uploadFiles(files, options = {}) {
    const fileArray = Array.from(files);
    const uploadPromises = [];
    const results = [];

    for (let i = 0; i < fileArray.length; i++) {
      const file = fileArray[i];
      const uploadPromise = this.uploadSingleFile(file, i, fileArray.length, options);
      uploadPromises.push(uploadPromise);
    }

    try {
      const uploadResults = await Promise.all(uploadPromises);
      return uploadResults;
    } catch (error) {
      console.error('多文件上传失败:', error);
      throw error;
    }
  }

  /**
   * 上传单个文件
   */
  uploadSingleFile(file, index, total, options = {}) {
    return new Promise((resolve, reject) => {
      const upload = new Upload(file, {
        endpoint: this.endpoint,
        retryDelays: [0, 1000, 3000, 5000],
        chunkSize: options.chunkSize || 1024 * 1024 * 2, // 2MB chunks
        metadata: {
          filename: file.name,
          filetype: file.type,
          ...options.metadata
        },

        onError: (error) => {
          console.error(`文件 ${file.name} 上传失败:`, error);
          this.uploads.delete(file.name);

          if (this.onError) {
            this.onError(error, file, index);
          }
          reject(error);
        },

        onProgress: (bytesUploaded, bytesTotal) => {
          const progress = {
            file: file.name,
            index,
            bytesUploaded,
            bytesTotal,
            percentage: ((bytesUploaded / bytesTotal) * 100).toFixed(2)
          };

          if (this.onProgress) {
            this.onProgress(progress, this.getAllProgress());
          }
        },

        onSuccess: () => {
          console.log(`文件 ${file.name} 上传成功, URL:`, upload.url);

          const result = {
            file: file.name,
            index,
            url: upload.url,
            size: file.size,
            type: file.type
          };

          this.uploads.delete(file.name);

          if (this.onComplete) {
            this.onComplete(result, index, total);
          }

          resolve(result);
        }
      });

      // 存储上传实例以便管理
      this.uploads.set(file.name, upload);

      // 开始上传
      upload.start();
    });
  }

  /**
   * 获取所有文件的上传进度
   */
  getAllProgress() {
    const allProgress = [];
    this.uploads.forEach((upload, filename) => {
      // 注意：这里需要手动跟踪进度，因为 upload 对象可能没有直接的进度属性
      allProgress.push({
        filename,
        // 可以在 onProgress 回调中存储进度信息
      });
    });
    return allProgress;
  }

  /**
   * 取消所有上传
   */
  cancelAll() {
    this.uploads.forEach((upload) => {
      upload.abort();
    });
    this.uploads.clear();
  }

  /**
   * 取消特定文件的上传
   */
  cancelFile(filename) {
    const upload = this.uploads.get(filename);
    if (upload) {
      upload.abort();
      this.uploads.delete(filename);
    }
  }
}

// 使用示例
const uploader = new TusMultiFileUploader('http://localhost:1080/files/');

// 设置回调函数
uploader.onProgress = (fileProgress, allProgress) => {
  console.log(`${fileProgress.file}: ${fileProgress.percentage}%`);
};

uploader.onComplete = (result, index, total) => {
  console.log(`文件 ${result.file} 完成上传 (${index + 1}/${total})`);
};

uploader.onError = (error, file, index) => {
  console.error(`文件 ${file.name} 上传失败:`, error);
};

// 开始上传
document.getElementById('fileInput').addEventListener('change', async (event) => {
  const files = event.target.files;
  if (files.length > 0) {
    try {
      const results = await uploader.uploadFiles(files, {
        chunkSize: 1024 * 1024 * 5, // 5MB chunks
        metadata: {
          author: 'user123'
        }
      });
      console.log('所有文件上传完成:', results);
    } catch (error) {
      console.error('上传过程中出现错误:', error);
    }
  }
});