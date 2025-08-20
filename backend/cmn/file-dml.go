package cmn

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/viper"
)

type FileDeleteTask struct {
	FilePath string
	Digest   string
}

var (
	fileDeleteChan = make(chan FileDeleteTask, 1000) // 缓冲1000个任务
	deleteWorkers  sync.Once
	workerCount    int = 5
)

func init() {
	initFileDeleteWorkers()
}

// 初始化文件删除工作池
func initFileDeleteWorkers() {
	deleteWorkers.Do(func() {
		// 启动工作协程
		for i := 0; i < workerCount; i++ {
			go fileDeleteWorker()
		}
	})
}

func deleteFileAsync(filePath, digest string) {
	infoFilePath := filePath + ".info"

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		z.Error(fmt.Sprintf("异步删除文件失败: %s, digest: %s, error: %v",
			filePath, digest, err))
	}

	err = os.Remove(infoFilePath)
	if err != nil && !os.IsNotExist(err) {
		z.Error(fmt.Sprintf("异步删除.info文件失败: %s, digest: %s, error: %v",
			infoFilePath, digest, err))
	}
}

func fileDeleteWorker() {
	for task := range fileDeleteChan {
		deleteFileAsync(task.FilePath, task.Digest)
	}
}

// NewFileRecord 创建文件记录
//
// 该函数实现文件记录的 UPSERT 操作：
// 1. 首先根据文件摘要、文件名、创建者和域ID查询文件记录是否已存在
// 2. 如果文件已存在，直接返回现有文件的ID，exists=true
// 3. 如果文件不存在，插入新的文件记录并返回新文件的ID，exists=false
//
// 参数：
//   - ctx: 上下文
//   - tx: pgx事务，如果不传入则会在内部创建事务
//   - fileDigest: 文件摘要/哈希值
//   - fileName: 文件名
//   - fileSize: 文件大小（字节）
//   - domainID: 域ID，标识文件所属的域
//   - creator: 创建者用户ID
//
// 返回值：
//
//   - exists: 文件是否已存在，true表示文件已存在，false表示是新插入的文件
//   - fileID: 文件记录的ID，无论是已存在还是新创建的文件
//   - err: 错误信息，操作成功时为nil
//
// 补充：
//   - 函数会自动生成文件存储路径（基于配置文件中的上传目录和文件摘要）
//
// 使用示例：
//
//	err, exists, fileID := NewFileRecord(ctx, tx, "abc123", "test.pdf", 1024, 1, 100)
//	if err != nil {
//	    // 处理错误
//	}
//	if exists {
//	    // 文件已存在，fileID是现有文件的ID
//	} else {
//	    // 文件是新创建的，fileID是新文件的ID
//	}
func NewFileRecord(ctx context.Context, tx pgx.Tx, fileDigest string, fileName string, fileSize int64, domainID int64, creator int64) (exists bool, fileID int64, err error) {

	var forceResult string
	if val := ctx.Value("NewFileRecord-force-result"); val != nil {
		forceResult = val.(string)
	}

	if forceResult == "returnTrue" {
		return true, 99999, nil
	}

	if forceResult == "returnFalse" {
		return false, 99999, nil
	}

	if forceResult == "returnError" {
		return false, 0, fmt.Errorf("强制返回错误")
	}

	if fileDigest == "" {
		err = fmt.Errorf("fileDigest 不能为空")
		z.Error(err.Error())
		return
	}

	if fileName == "" {
		err = fmt.Errorf("fileName 不能为空")
		z.Error(err.Error())
		return
	}

	if fileSize <= 0 {
		err = fmt.Errorf("fileSize 不能为空")
		z.Error(err.Error())
		return
	}

	if domainID <= 0 {
		err = fmt.Errorf("domainID 不能为空")
		z.Error(err.Error())
		return
	}

	if creator <= 0 {
		err = fmt.Errorf("creator 不能为空")
		z.Error(err.Error())
		return
	}

	if tx == nil {
		conn := GetPgxConn()
		tx, err = conn.Begin(ctx)
		if forceResult == "tx.Begin" {
			err = fmt.Errorf("强制开始事务错误")
		}
		if err != nil {
			z.Error(err.Error())
			return
		}

		defer func() {
			if err != nil || forceResult == "tx.Rollback" {
				rErr := tx.Rollback(ctx)
				if forceResult == "tx.Rollback" {
					rErr = fmt.Errorf("强制回滚事务错误")
				}
				if rErr != nil {
					z.Error(rErr.Error())
				}
				return
			}

			cErr := tx.Commit(ctx)
			if forceResult == "tx.Commit" {
				cErr = fmt.Errorf("强制提交事务错误")
			}
			if cErr != nil {
				z.Error(cErr.Error())
			}
		}()
	}

	exists = true
	z.Info(forceResult)

	// 查询是否已存在
	err = tx.QueryRow(ctx, `
			SELECT id FROM t_file
			WHERE digest = $1 
			AND file_name = $2 
			AND creator = $3
			AND domain_id = $4
			LIMIT 1
		`, fileDigest, fileName, creator, domainID).Scan(&fileID)

	// 如果查询失败，则标记为新文件
	if err == pgx.ErrNoRows {
		exists = false
	}
	if forceResult == "tx.QueryRow1" {
		err = fmt.Errorf("强制查询文件错误")
	}
	if err != pgx.ErrNoRows && err != nil {
		err = fmt.Errorf("查询文件失败: %v", err)
		z.Error(err.Error())
		return
	}

	// 如果该文件已存在，则返回该文件ID，不做任何操作
	if exists {
		return
	}

	key := "tusd.fileStorePath"
	uploadDir := "./uploads"
	if viper.IsSet(key) {
		uploadDir = viper.GetString(key)
	}
	uploadDir, err = filepath.Abs(uploadDir)
	if forceResult == "filepath.Abs" {
		err = fmt.Errorf("强制获取上传目录绝对路径错误")
	}
	if err != nil {
		err = fmt.Errorf("获取上传目录绝对路径失败: %v", err)
		z.Error(err.Error())
		return
	}

	var filePath string
	filePath = filepath.Join(uploadDir, fmt.Sprintf("%s", fileDigest))

	var currentTime = time.Now().UnixMilli()

	// 如果该文件不存在，则插入新记录
	err = tx.QueryRow(ctx, `
		INSERT INTO t_file (digest, file_name, path, belongto_path, size, count, creator, create_time, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, fileDigest, fileName, filePath, filePath, fileSize, 1, creator, currentTime, "0").Scan(&fileID)
	if forceResult == "tx.QueryRow2" {
		err = fmt.Errorf("强制插入文件信息错误")
	}
	if err != nil {
		err = fmt.Errorf("插入文件信息失败: %v", err)
		z.Error(err.Error())
		return
	}

	return
}

// ChangeFileReferenceCount 修改文件引用计数
//
// 该函数用于管理文件记录的引用计数，支持增加或减少引用计数：
// 1. 当 count > 0 时，增加文件的引用计数
// 2. 当 count < 0 时，减少文件的引用计数
// 3. 当引用计数减少到 0 或以下时，删除文件记录
// 4. 当没有其他相同 digest 的文件时，从文件系统中删除物理文件
//
// 参数：
//   - ctx: 上下文
//   - tx: 数据库事务，如果为 nil 则函数内部会创建新事务
//   - fileID: 要修改引用计数的文件ID
//   - count: 引用计数变化量，正数表示增加，负数表示减少，不能为0
//
// 返回值：
//   - err: 错误信息，操作成功时为 nil
//
// 处理逻辑：
//   - count > 0: 直接增加 t_file 表中的 count 字段
//   - count < 0:
//   - 如果剩余引用数 > count，则减少引用计数
//   - 如果剩余引用数 <= count，则删除文件记录
//   - 检查是否还有相同 digest 的其他文件
//   - 如果没有，则从文件系统删除物理文件和对应的 .info 文件
//
// 补充：
//   - 该函数可能涉及文件删除操作，不可回滚，尽量在业务逻辑的末端使用
//   - 该函数的文件删除为异步操作，不会阻塞，同时如果删除文件报错也不会返回错误
//
// 使用示例：
//
//	// 增加引用计数
//	err := ChangeFileReferenceCount(ctx, tx, fileID, 1)
//
//	// 减少引用计数
//	err := ChangeFileReferenceCount(ctx, tx, fileID, -1)
//
//	// 减少多个引用
//	err := ChangeFileReferenceCount(ctx, tx, fileID, -3)
func ChangeFileReferenceCount(ctx context.Context, tx pgx.Tx, fileID int64, count int64) (err error) {

	var forceResult string
	if val := ctx.Value("ChangeFileReferenceCount-force-result"); val != nil {
		forceResult = val.(string)
	}

	if forceResult == "returnError" {
		return fmt.Errorf("强制返回错误")
	}

	if forceResult == "returnNil" {
		return nil
	}

	if fileID <= 0 {
		err = fmt.Errorf("fileID 不能为空")
		z.Error(err.Error())
		return
	}

	if count == 0 {
		err = fmt.Errorf("count 不能为空")
		z.Error(err.Error())
		return
	}

	if tx == nil {
		conn := GetPgxConn()
		tx, err = conn.Begin(ctx)
		if forceResult == "tx.Begin" {
			err = fmt.Errorf("强制开始事务错误")
		}
		if err != nil {
			z.Error(err.Error())
			return
		}

		defer func() {
			if err != nil || forceResult == "tx.Rollback" {
				rErr := tx.Rollback(ctx)
				if forceResult == "tx.Rollback" {
					rErr = fmt.Errorf("强制回滚事务错误")
				}
				if rErr != nil {
					z.Error(rErr.Error())
				}
				return
			}

			cErr := tx.Commit(ctx)
			if forceResult == "tx.Commit" {
				cErr = fmt.Errorf("强制提交事务错误")
			}
			if cErr != nil {
				z.Error(cErr.Error())
			}
		}()
	}

	// 如果变化的引用计数大于0，则增加t_file表中的引用计数
	if count > 0 {
		_, err = tx.Exec(ctx, `
			UPDATE t_file SET count = count + $1 WHERE id = $2
		`, count, fileID)
		if forceResult == "tx.UpdateCount" {
			err = fmt.Errorf("强制更新文件引用计数错误")
		}
		if err != nil {
			err = fmt.Errorf("更新文件引用计数失败: %v", err)
			z.Error(err.Error())
			return
		}

		return
	}

	// 如果变化的引用计数小于0，则根据t_file表中的引用计数进行判断
	var refCount int64
	var digest, filePath string
	err = tx.QueryRow(ctx, "SELECT count, digest, path FROM t_file WHERE id = $1", fileID).Scan(
		&refCount, &digest, &filePath)
	if forceResult == "tx.QueryRow" {
		err = fmt.Errorf("强制查询文件信息错误")
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	// 如果剩余的引用次数多余要减少的引用计数，则直接更新
	if refCount > -count {
		_, err = tx.Exec(ctx, "UPDATE t_file SET count = count + $1 WHERE id = $2", count, fileID)
		if forceResult == "tx.UpdateCount" {
			err = fmt.Errorf("强制更新文件引用计数错误")
		}
		if err != nil {
			z.Error(err.Error())
			return err
		}

		return
	}

	// 如果剩余引用次数不足，则删除该文件记录
	_, err = tx.Exec(ctx, "DELETE FROM t_file WHERE id = $1", fileID)
	if forceResult == "tx.DeleteFile" {
		err = fmt.Errorf("强制删除文件记录错误")
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	// 检查是否还有其他相同digest的文件记录
	var sameDigest int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM t_file WHERE digest = $1", digest).Scan(&sameDigest)
	if forceResult == "tx.CountDigest" {
		err = fmt.Errorf("强制统计相同digest文件错误")
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	// 如果没有其他相同digest的文件记录，从文件系统删除该文件
	if sameDigest == 0 {
		select {
		case fileDeleteChan <- FileDeleteTask{
			FilePath: filePath,
			Digest:   digest,
		}:
			z.Info(fmt.Sprintf("文件删除任务已添加: %s", filePath))
		default:
			z.Warn(fmt.Sprintf("文件删除队列已满，跳过删除: %s", filePath))
		}
	}

	return
}
