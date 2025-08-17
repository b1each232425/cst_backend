# patch ```tus-js-cln/lib/upload.ts```

conjunction with tusd-2.8.0

2025-08-13 09:55:00 by kzz
## js-tus-client 4.3.1

## web/src/lib/tus-js-cln/lib/upload.ts
after L654

```diff
--- a/web/src/lib/tus-js-cln/lib/upload.ts
+++ b/web/src/lib/tus-js-cln/lib/upload.ts
@@ -651,6 +651,21 @@
    if (typeof this.options.onUploadUrlAvailable === 'function') {
       await this.options.onUploadUrlAvailable()
    }
+   //-----------------------------------------
+   // for exists upload process 
+   // 208 means already exists
+   //StatusAlreadyReported      = 208 // RFC 5842, 7.1
+   if (res.getStatus() == 208) {
+     this._emitSuccess(res)
+     return;
+   }
+   //-------------------------------
    if (this._size === 0) {
      // Nothing to upload and file was successfully created
      await this._emitSuccess(res)
      if (this._source) this._source.close()
      return
    }

    await this._saveUploadInUrlStorage()

```