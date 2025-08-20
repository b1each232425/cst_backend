package cmn

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	fileID1 int64 = 99990
	fileID2 int64 = 99991
	fileID3 int64 = 99992
	fileID4 int64 = 99993

	fileID1Digest string = "digest1"
	fileID2Digest string = "digest2"
	fileID3Digest string = "digest3"
	fileID4Digest string = "digest4"

	fileID1Name string = "file1.txt"
	fileID2Name string = "file2.txt"
	fileID3Name string = "file3.txt"
	fileID4Name string = "file4.txt"

	fileID1Size int64 = 1024
	fileID2Size int64 = 2048
	fileID3Size int64 = 3072
	fileID4Size int64 = 4096
)

func createTestData(t *testing.T) {

	// 创建测试数据

	conn := GetPgxConn()

	ctx := context.Background()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("事务回滚: %v", r)
		} else {
			if err != nil {
				tx.Rollback(ctx)
				t.Logf("事务回滚: %v", err)
			} else {
				err = tx.Commit(ctx)
			}
		}
	}()

	uploadDir := "./uploads"
	fileID1Path := filepath.Join(uploadDir, fileID1Digest)
	fileID2Path := filepath.Join(uploadDir, fileID2Digest)
	fileID3Path := filepath.Join(uploadDir, fileID3Digest)
	fileID4Path := filepath.Join(uploadDir, fileID4Digest)

	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, 2000, '0')
        RETURNING id
    `, fileID1, fileID1Digest, fileID1Name, fileID1Path, "/test/files", fileID1Size, 99999, time.Now().UnixMilli()).Scan(&fileID1)
	if err != nil {
		t.Fatalf("插入测试文件1记录失败: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 2, $7, $8, 2000, '0')
        RETURNING id
    `, fileID2, fileID2Digest, fileID2Name, fileID2Path, "/test/files", fileID2Size, 99999, time.Now().UnixMilli()).Scan(&fileID2)
	if err != nil {
		t.Fatalf("插入测试文件2记录失败: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, 2000, '0')
        RETURNING id
    `, fileID3, fileID3Digest, fileID3Name, fileID3Path, "/test/files", fileID3Size, 99999, time.Now().UnixMilli()).Scan(&fileID3)
	if err != nil {
		t.Fatalf("插入测试文件3记录失败: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO t_file (id, digest, file_name, path, belongto_path, size, count, creator, create_time, domain_id, status)
        VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, 2000, '0')
        RETURNING id
    `, fileID4, fileID4Digest, fileID4Name, fileID4Path, "/test/files", fileID4Size, 99999, time.Now().UnixMilli()).Scan(&fileID4)
	if err != nil {
		t.Fatalf("插入测试文件4记录失败: %v", err)
	}

	return
}

func cleanTestData(t *testing.T) {
	conn := GetPgxConn()

	ctx := context.Background()

	// 开始事务
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Logf("开始清理事务失败: %v", err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			t.Logf("事务回滚: %v", r)
		} else {
			if err != nil {
				tx.Rollback(ctx)
				t.Logf("事务回滚: %v", err)
			} else {
				tx.Commit(ctx)
			}
		}
	}()

	_, err = tx.Exec(ctx, `
        DELETE FROM t_file WHERE id = ANY($1)
    `, []int64{fileID1, fileID2, fileID3, fileID4})
	if err != nil {
		t.Logf("删除测试文件记录失败: %v", err)
	}

	_, err = tx.Exec(ctx, `
        DELETE FROM t_file WHERE creator = $1
    `, 99999)
	if err != nil {
		t.Logf("删除测试文件记录失败: %v", err)
	}

	return
}

func Test_initFileDeleteWorkers(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initFileDeleteWorkers()
		})
	}
}

func TestNewFileRecord(t *testing.T) {

	ConfigureForTest()

	cleanTestData(t)
	createTestData(t)
	t.Cleanup(func() {
		cleanTestData(t)
	})

	type args struct {
		ctx        context.Context
		tx         pgx.Tx
		fileDigest string
		fileName   string
		fileSize   int64
		domainID   int64
		creator    int64
	}
	tests := []struct {
		name       string
		args       args
		wantExists bool
		wantFileID int64
		wantErr    bool
	}{
		{
			name: "参数验证 - fileDigest为空",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: "",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "参数验证 - fileName为空",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "参数验证 - fileSize小于等于0",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   0,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "参数验证 - domainID小于等于0",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   0,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "参数验证 - creator小于等于0",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    0,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "成功创建新文件记录",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: "newdigest123",
				fileName:   "newfile.txt",
				fileSize:   2048,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0, // 我们不知道具体的ID，只验证不为0
			wantErr:    false,
		},
		{
			name: "文件已存在 - 返回现有文件ID",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: fileID1Digest,
				fileName:   fileID1Name,
				fileSize:   fileID1Size,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: true,
			wantFileID: fileID1,
			wantErr:    false,
		},
		{
			name: "不同creator的相同文件 - 创建新记录",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: fileID1Digest,
				fileName:   fileID1Name,
				fileSize:   fileID1Size,
				domainID:   2000,
				creator:    88888,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    false,
		},
		{
			name: "不同domainID的相同文件 - 创建新记录",
			args: args{
				ctx:        context.Background(),
				tx:         nil,
				fileDigest: fileID1Digest,
				fileName:   fileID1Name,
				fileSize:   fileID1Size,
				domainID:   3000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    false,
		},
		// ForceResult 测试用例
		{
			name: "ForceResult - returnTrue",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "returnTrue"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: true,
			wantFileID: 99999,
			wantErr:    false,
		},
		{
			name: "ForceResult - returnFalse",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "returnFalse"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 99999,
			wantErr:    false,
		},
		{
			name: "ForceResult - returnError",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "returnError"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "ForceResult - tx.Begin错误",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "tx.Begin"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "ForceResult - tx.Rollback错误",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "tx.Rollback"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    false,
		},
		{
			name: "ForceResult - tx.Commit错误",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "tx.Commit"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    false,
		},
		{
			name: "ForceResult - tx.QueryRow1错误",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "tx.QueryRow1"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "ForceResult - tx.QueryRow2错误",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "tx.QueryRow2"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
		{
			name: "ForceResult - filepath.Abs错误",
			args: args{
				ctx:        context.WithValue(context.Background(), "NewFileRecord-force-result", "filepath.Abs"),
				tx:         nil,
				fileDigest: "testdigest123",
				fileName:   "test.txt",
				fileSize:   1024,
				domainID:   2000,
				creator:    99999,
			},
			wantExists: false,
			wantFileID: 0,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExists, gotFileID, err := NewFileRecord(tt.args.ctx, tt.args.tx, tt.args.fileDigest, tt.args.fileName, tt.args.fileSize, tt.args.domainID, tt.args.creator)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileRecord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotExists != tt.wantExists {
				t.Errorf("NewFileRecord() gotExists = %v, want %v", gotExists, tt.wantExists)
			}

			if tt.wantErr {
				return
			}

			// 对于新创建的文件，我们只验证fileID不为0
			if !tt.wantErr && tt.wantFileID == 0 {
				if gotFileID == 0 {
					t.Errorf("NewFileRecord() gotFileID = %v, expected non-zero for new file", gotFileID)
				}
			} else if gotFileID != tt.wantFileID {
				t.Errorf("NewFileRecord() gotFileID = %v, want %v", gotFileID, tt.wantFileID)
			}

			// 清理新创建的测试记录
			if !tt.wantErr && gotFileID > 0 && !tt.wantExists {
				conn := GetPgxConn()
				ctx := context.Background()
				_, cleanupErr := conn.Exec(ctx, "DELETE FROM t_file WHERE id = $1", gotFileID)
				if cleanupErr != nil {
					t.Logf("清理新创建的文件记录失败: %v", cleanupErr)
				}
			}
		})
	}
}

func TestChangeFileReferenceCount(t *testing.T) {

	ConfigureForTest()

	cleanTestData(t)
	createTestData(t)
	t.Cleanup(func() {
		cleanTestData(t)
	})

	type args struct {
		ctx    context.Context
		tx     pgx.Tx
		fileID int64
		count  int64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "参数验证 - fileID小于等于0",
			args: args{
				ctx:    context.Background(),
				tx:     nil,
				fileID: 0,
				count:  1,
			},
			wantErr: true,
		},
		{
			name: "参数验证 - count为0",
			args: args{
				ctx:    context.Background(),
				tx:     nil,
				fileID: fileID1,
				count:  0,
			},
			wantErr: true,
		},
		{
			name: "成功增加引用计数",
			args: args{
				ctx:    context.Background(),
				tx:     nil,
				fileID: fileID1,
				count:  1,
			},
			wantErr: false,
		},
		{
			name: "成功减少引用计数(不删除)",
			args: args{
				ctx:    context.Background(),
				tx:     nil,
				fileID: fileID2, // 测试数据中fileID2的count为2
				count:  -1,
			},
			wantErr: false,
		},
		{
			name: "减少引用计数导致删除文件记录",
			args: args{
				ctx:    context.Background(),
				tx:     nil,
				fileID: fileID3, // 测试数据中fileID3的count为1
				count:  -1,
			},
			wantErr: false,
		},
		// ForceResult 测试用例
		{
			name: "ForceResult - returnError",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "returnError"),
				tx:     nil,
				fileID: fileID1,
				count:  1,
			},
			wantErr: true,
		},
		{
			name: "ForceResult - returnNil",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "returnNil"),
				tx:     nil,
				fileID: fileID1,
				count:  1,
			},
			wantErr: false,
		},
		{
			name: "ForceResult - tx.Begin错误",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "tx.Begin"),
				tx:     nil,
				fileID: fileID1,
				count:  1,
			},
			wantErr: true,
		},
		{
			name: "ForceResult - tx.Rollback错误",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "tx.Rollback"),
				tx:     nil,
				fileID: fileID1,
				count:  1,
			},
			wantErr: false,
		},
		{
			name: "ForceResult - tx.Commit错误",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "tx.Commit"),
				tx:     nil,
				fileID: fileID1,
				count:  1,
			},
			wantErr: false,
		},
		{
			name: "ForceResult - tx.QueryRow错误",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "tx.QueryRow"),
				tx:     nil,
				fileID: fileID1,
				count:  -1,
			},
			wantErr: true,
		},
		{
			name: "ForceResult - tx.UpdateCount错误(增加计数)",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "tx.UpdateCount"),
				tx:     nil,
				fileID: fileID1,
				count:  1, // 正数，会触发增加引用计数的UpdateCount
			},
			wantErr: true,
		},
		{
			name: "ForceResult - tx.UpdateCount错误(减少计数)",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "tx.UpdateCount"),
				tx:     nil,
				fileID: fileID2, // 测试数据中fileID2的count为2，减1后不会删除
				count:  -1,
			},
			wantErr: true,
		},
		{
			name: "ForceResult - tx.DeleteFile错误",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "tx.DeleteFile"),
				tx:     nil,
				fileID: fileID4, // 测试数据中fileID4的count为1，减1后会删除
				count:  -1,
			},
			wantErr: true,
		},
		{
			name: "ForceResult - tx.CountDigest错误",
			args: args{
				ctx:    context.WithValue(context.Background(), "ChangeFileReferenceCount-force-result", "tx.CountDigest"),
				tx:     nil,
				fileID: fileID4, // 先创建一个新的测试文件ID
				count:  -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 对于会删除文件的测试，需要重新创建测试数据
			if tt.name == "减少引用计数导致删除文件记录" {
				cleanTestData(t)
				createTestData(t)
			}
			if tt.name == "ForceResult - tx.DeleteFile错误" || tt.name == "ForceResult - tx.CountDigest错误" {
				cleanTestData(t)
				createTestData(t)
			}

			err := ChangeFileReferenceCount(tt.args.ctx, tt.args.tx, tt.args.fileID, tt.args.count)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangeFileReferenceCount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestChangeFileReferenceCount_QueueFull 测试文件删除队列已满的情况
func TestChangeFileReferenceCount_QueueFull(t *testing.T) {
	ConfigureForTest()
	cleanTestData(t)
	createTestData(t)
	t.Cleanup(func() {
		cleanTestData(t)
	})

	// 备份原始队列
	originalChan := fileDeleteChan

	// 创建一个容量为0的队列，这样任何发送操作都会立即触发default分支
	fileDeleteChan = make(chan FileDeleteTask, 0)

	// 恢复原始队列
	defer func() {
		close(fileDeleteChan) // 关闭测试队列
		fileDeleteChan = originalChan
	}()

	// 测试：减少引用计数导致删除文件记录，此时队列已满会触发default分支
	err := ChangeFileReferenceCount(context.Background(), nil, fileID4, -1)
	if err != nil {
		t.Errorf("ChangeFileReferenceCount() error = %v, wantErr false", err)
	}
}

func Test_deleteFileAsync(t *testing.T) {
	ConfigureForTest()

	// 创建测试目录
	testDir := "./test_files"
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	defer os.RemoveAll(testDir) // 清理测试目录

	// 测试1: 删除不存在的文件(不会产生错误日志)
	t.Run("删除不存在的文件(不会产生错误日志)", func(t *testing.T) {
		deleteFileAsync("./test_files/nonexistent_file.txt", "test_digest_1")
		// 验证文件确实不存在
		if _, err := os.Stat("./test_files/nonexistent_file.txt"); !os.IsNotExist(err) {
			t.Errorf("文件应该不存在")
		}
	})

	// 测试2: 删除正在使用中的文件(会产生错误日志)
	t.Run("删除正在使用中的文件(会产生错误日志)", func(t *testing.T) {
		// 创建一个文件并保持打开状态
		lockedFile := "./test_files/locked_file.txt"
		file, err := os.Create(lockedFile)
		if err != nil {
			t.Fatalf("创建锁定测试文件失败: %v", err)
		}

		// 写入一些内容
		_, err = file.WriteString("test content")
		if err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}

		// 不关闭文件，保持锁定状态
		defer file.Close() // 确保测试结束后关闭

		// 尝试删除正在使用的文件
		deleteFileAsync(lockedFile, "test_digest_2")
	})

	// 测试3: 删除正在使用中的.info文件
	t.Run("删除正在使用中的.info文件(会产生错误日志)", func(t *testing.T) {
		// 创建一个普通文件（可以成功删除）
		normalFile := "./test_files/normal_file.txt"
		err := os.WriteFile(normalFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("创建普通测试文件失败: %v", err)
		}

		// 创建一个.info文件并保持打开状态
		infoFile := normalFile + ".info"
		file, err := os.Create(infoFile)
		if err != nil {
			t.Fatalf("创建锁定的.info测试文件失败: %v", err)
		}

		// 写入一些内容
		_, err = file.WriteString("info content")
		if err != nil {
			t.Fatalf("写入.info测试文件失败: %v", err)
		}

		// 不关闭文件，保持锁定状态
		defer file.Close() // 确保测试结束后关闭

		// 尝试删除，主文件应该成功，.info文件应该失败并产生错误日志
		deleteFileAsync(normalFile, "test_digest_3")
	})
}
