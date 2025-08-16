<script>
	/* using for minimap
		http://patorjk.com/software/taag/#p=display&f=Roman&t=code
    ===============================================================================   

                                                 .o8            
                                                "888            
                         .ooooo.   .ooooo.   .oooo888   .ooooo.  
                        d88' `"Y8 d88' `88b d88' `888  d88' `88b 
                        888       888   888 888   888  888ooo888 
                        888   .o8 888   888 888   888  888    .o 
                        `Y8bod8P' `Y8bod8P' `Y8bod88P" `Y8bod8P'        

    ===============================================================================
	*/
	import { onMount } from 'svelte';
	import { createXXHash64 } from 'hash-wasm';
	import { filesize } from 'filesize';
	// import { Upload as tus } from '$lib/tus-js-cln/lib/browser';
	// import * as tus from 'tus-js-client';
	let endpoint = $state('http://localhost:8080/api/file');
	const CHUNKSIZE = 1024 * 1024 * 4;
	let chunkSize = $state(CHUNKSIZE);
	let parallelUploads = $state(1);

	let selectedFiles = $state();
	let clearSelectedFiles = () => {
		selectedFiles = new DataTransfer().files;
	};

	let jobs = $state(new Map());
	let tus;

	let fastdigest = (job) => {
		return new Promise(async (resolve, reject) => {
			if (!job || !job.file) {
				reject('invalid/null job');
				return;
			}

			let md = await createXXHash64();
			md.init();

			let fileReader = new FileReader();

			let read = 0;
			fileReader.onload = (e) => {
				if (!e || !e.target || !e.target.result) {
					let err = new Error('invalid event.target.result');
					console.log(err);
					jobs.delete(job.id);
					reject(err);
					return;
				}

				read += e.target.result.byteLength;
				let buf = new Uint8Array(e.target.result);
				md.update(buf);
				seek();
			};

			let fileSize = job.file.size;
			let start = 0,
				end = 0;

			let seek = () => {
				let now = new Date();
				job.sumPerformance =
					(((read * 1.0) / (now.getTime() - beginTime.getTime())) * 1000) / (1024 * 1024);

				job.sumProgress = (((read * 1.0) / fileSize) * 100).toFixed(2);
				if (read >= fileSize) {
					let hex = md.digest();
					resolve(hex);
					return;
				}

				end += CHUNKSIZE;
				end = end < fileSize ? end : fileSize + 1;
				let slice = job.file.slice(start, end);

				fileReader.readAsArrayBuffer(slice);
				start = end;
			};

			let beginTime = new Date();
			seek();
		});
	};

	function encodeMetadata(metadata) {
		const encodedPairs = [];

		for (const [key, value] of Object.entries(metadata)) {
			// 将值转换为字符串并进行 base64 编码
			const encodedValue = btoa(unescape(encodeURIComponent(String(value))));
			encodedPairs.push(`${key} ${encodedValue}`);
		}

		return encodedPairs.join(',');
	}

	async function singles(job) {
		return new Promise(async (resolve, reject) => {
			if (!job || !job.file) {
				reject('invalid/null job');
				return;
			}

			job.checksum = await fastdigest(job);
			let metadata = {
				filename: job.file.name,
				filetype: job.file.type,
				filesize: job.file.size,
				lastModified: job.file.lastModified,
				checksum: job.checksum,
			};

			const encodedMetadata = encodeMetadata(metadata);
			let v = encodeURIComponent(encodedMetadata);
			// console.log(v);
			const tusOptions = {
				endpoint: `${endpoint}?metadata=${v}`,
				chunkSize,
				retryDelays: [0, 1000, 3000, 5000],
				parallelUploads,
				metadata,
				onUploadUrlAvailable() {
					job.url = job.tus.url;
				},
				onError(error) {
					console.log(error);
					reject(error);
				},
				onProgress(bytesUploaded, bytesTotal) {
					job.transmitPercentage = ((bytesUploaded / bytesTotal) * 100).toFixed(2);
					job.bytesUploaded = bytesUploaded;
					job.bytesTotal = bytesTotal;
				},
				onSuccess(resp) {
					// let x = resp.lastResponse._xhr;
					// let msg = `上传成功`;
					// if (x.status === 208) {
					// 	msg = '文件已经在服务器上了';
					// }
					// console.log(`${metadata.filename} ${msg}(${x.status}): ${job.url}`);

					resolve(job);
				},
			};

			job.tus = new tus.Upload(job.file, tusOptions);
			job.tus.start();
		});
	}

	async function uploadFiles(files = selectedFiles) {
		let promises = [];
		for (let i = 0; i < files.length; i++) {
			const file = files[i];
			if (!file) {
				continue;
			}

			let id = `${file.name}#${file.size}#${file.lastModified}`;
			let job = { id, file };
			jobs.set(id, job);

			const p = singles(job);
			promises.push(p);
		}

		let results;
		try {
			// var results: [job]
			// job:{ID,file,url}
			results = await Promise.all(promises);
			results.forEach((e) => {
				console.log(`download: ${e.file.name}: ${e.url}`);
			});
		} catch (err) {
			console.log(err);
		}

		reset();
		queryFiles();
	}

	function reset() {
		clearSelectedFiles();
	}

	function tusInit() {
		if (!tus || !tus.isSupported) {
			console.log('tus unsupported');
			return;
		}
	}

	let fileApi = '/api/file';
	let criteria = $state('.*');
	let uploadedFiles = $state([]);

	let queryFiles = () => {
		let v = encodeURIComponent(criteria);
		fetch(fileApi + `/nonexistence?q=${v}`)
			.then((v) => {
				let size = v.headers.get('content-length');
				if (!v || size === '0') {
					return [];
				}

				return v.json();
			})
			.then((v) => {
				if (!v || v.length == 0) {
					console.log('empty file list');
					return;
				}

				let d = [];
				for (let i = 0; i < v.length; i++) {
					let metadata = v[i].MetaData;

					// metadata.full = v[i];
					metadata.url = `${fileApi}/${v[i].ID}`;
					if (!metadata.filename) {
						metadata.filename = v[i].ID;
					}

					if (!metadata.filesize) {
						metadata.filesize = v[i].Size;
					}

					if (!metadata.checksum) {
						metadata.checksum = v[i].ID;
					}

					d.push(metadata);
				}
				uploadedFiles = d;
			})
			.catch((err) => {
				console.log(err);
			});
	};

	let removeFile = (e, metaData) => {
		if (e && e.preventDefault) {
			e.preventDefault();
		}

		if (!metaData || !metaData.checksum) {
			console.log('invalid/null metaData');
			return;
		}

		fetch(`${fileApi}/${metaData.checksum}`, {
			method: 'DELETE',
			headers: {
				'Tus-Resumable': '1.0.0',
			},
		})
			.then((v) => {
				if (v.status !== 204) {
					console.log(`delete ${metaData.filename} failed`);
					return;
				}

				console.log(`成功删除${metaData.filename}`);

				let idx = uploadedFiles.findIndex((e) => e.checksum === metaData.checksum);
				if (idx < 0) {
					console.log(`find ${metaData.checksum} failed`);
					return;
				}

				uploadedFiles.splice(idx, 1);
			})
			.catch((err) => console.log(err));
	};

	onMount(async () => {
		tus = await import('$lib/tus-js-cln/lib/browser');
		tusInit();
		queryFiles();
	});
</script>

<!-- using for minimap
    http://patorjk.com/software/taag/#p=display&f=Roman&t=template

                          .                                          oooo                .             
                        .o8                                          `888              .o8             
                      .o888oo  .ooooo.  ooo. .oo.  .oo.   oo.ooooo.   888   .oooo.   .o888oo  .ooooo.  
                        888   d88' `88b `888P"Y88bP"Y88b   888' `88b  888  `P  )88b    888   d88' `88b 
                        888   888ooo888  888   888   888   888   888  888   .oP"888    888   888ooo888 
                        888 . 888    .o  888   888   888   888   888  888  d8(  888    888 . 888    .o 
                        "888" `Y8bod8P' o888o o888o o888o  888bod8P' o888o `Y888""8o   "888" `Y8bod8P' 
                                                          888                                         
                                                          o888o                                        
  ============================================================================================================
-->
<main>
	<field>
		<span class="title">trial</span>
	</field>

	<field>
		<label>
			<span class="title">upload endpoint:</span>
			<input type="text" bind:value={endpoint} />
		</label>
	</field>

	<field>
		<label>
			<span class="title">chunk size (bytes):</span>
			<input type="number" bind:value={chunkSize} />
		</label>
	</field>
	<field>
		<label>
			<span class="title">parallel upload requests:</span>
			<input type="number" bind:value={parallelUploads} />
		</label>
	</field>

	<field>
		<label>
			<span class="title">criteria:</span>
			<input type="text" bind:value={criteria} />
		</label>
	</field>

	<field>
		<label>
			<span class="title">select target file(s):</span>
			<input type="file" multiple bind:files={selectedFiles} onchange={(e) => uploadFiles()} />
		</label>
	</field>

	<control-panel>
		<div class="uploadeds">
			<ul class="files">
				{#each uploadedFiles as file, i (file.checksum)}
					<li class="file {i % 2 == 0 ? 'even' : ''}">
						<a href={file.url} target="_blank" data-sveltekit-reload
							><div class="file-detail">
								<div class="filename">{file.filename}</div>
								<div class="fileinfo">
									<div class="filesize">{filesize(file.filesize, { standard: 'jedec' })}</div>
									<div class="create-time">
										{file.lastModified
											? new Date(parseInt(file.lastModified))
													.toISOString()
													.substring(0, 19)
													.replace('T', ' ')
											: ''}
									</div>
									<button onclick={(e) => removeFile(e, file)}>删 除</button>
								</div>
							</div></a
						>
					</li>
				{/each}
			</ul>
		</div>
	</control-panel>
</main>

<style lang="scss" scoped>
	/* using for minimap
  ===============================================================================    
                                  .               oooo            
                                .o8               `888            
                       .oooo.o .o888oo oooo    ooo  888   .ooooo.  
                      d88(  "8   888    `88.  .8'   888  d88' `88b 
                      `"Y88b.    888     `88..8'    888  888ooo888 
                      o.  )88b   888 .    `888'     888  888    .o 
                      8oo888P'   "888"     .8'     o888o `Y8bod8P' 
                                      .o..P'                      
                                      `Y8P'      
  ===============================================================================                                      
  */
	field {
		display: flex;
		padding: 0.4em;

		.title {
			display: inline-block;
			text-align: right;
			min-width: 10rem;
			color: blueviolet;
		}

		input {
			color: blue;
			&[type='text'] {
				min-width: 18em;
			}
		}
	}

	.uploadeds {
		display: flex;
		flex-flow: column wrap;
		// row-gap: 0.8em;
		border: 1px dotted grey;
		padding: 0 1em;

		ul.files {
			li.file {
				list-style: none;
				padding: 0.3em 0;

				&.even {
					background-color: lightgrey;
				}
				.file-detail {
					display: flex;
					flex-flow: row wrap;
					justify-content: space-between;

					.fileinfo {
						display: flex;
						flex-flow: row wrap;
						column-gap: 1em;
					}
				}
			}
		}
	}
</style>
