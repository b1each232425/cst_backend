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

func deleteFileAsync(ctx context.Context, filePath, digest string) {
	infoFilePath := filePath + ".info"

	var forceResult string
	if val := ctx.Value("deleteFileAsync-force-result"); val != nil {
		forceResult = val.(string)
	}

	conn := GetPgxConn()
	var count int
	err := conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM t_file WHERE digest = $1", digest).Scan(&count)
	if forceResult == "QueryRow" {
		err = fmt.Errorf("强制查询文件记录错误")
	}
	if err != nil {
		z.Error(fmt.Sprintf("检查digest是否还存在失败: %s, digest: %s, error: %v", filePath, digest, err))
		return
	}

	// 有其他引用，不删物理文件
	if count > 0 {
		z.Info(fmt.Sprintf("文件未删除，digest仍有引用: %s", filePath))
		return
	}

	err = os.Remove(filePath)
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
		deleteFileAsync(context.Background(), task.FilePath, task.Digest)
	}
}

// NewFileRecord 创建文件记录
//
// 该函数实现文件记录的 INSERT 操作：
//   - 插入新的文件记录并返回新文件的ID
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
//   - fileID: 文件记录的ID，无论是已存在还是新创建的文件
//   - err: 错误信息，操作成功时为nil
//
// 补充：
//   - 函数会自动生成文件存储路径（基于配置文件中的上传目录和文件摘要）
//
// 使用示例：
//
//	fileID, err := NewFileRecord(ctx, tx, "abc123", "test.pdf", 1024, 1, 100)
//	if err != nil {
//	    // 处理错误
//	}
func NewFileRecord(ctx context.Context, tx pgx.Tx, fileDigest string, fileName string, fileSize int64, domainID int64, creator int64) (fileID int64, err error) {

	var forceResult string
	if val := ctx.Value("NewFileRecord-force-result"); val != nil {
		forceResult = val.(string)
	}

	if forceResult == "returnTrue" {
		return 99999, nil
	}

	if forceResult == "returnFalse" {
		return 99999, nil
	}

	if forceResult == "returnError" {
		return 0, fmt.Errorf("强制返回错误")
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

	// 插入新记录
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

// DeleteFileRecord 删除文件记录并处理物理文件删除任务
//
// 该函数用于彻底删除指定文件ID的数据库记录，并在无其他引用时异步删除物理文件。
//
// 删除流程：
//  1. 查询文件摘要和存储路径。
//  2. 删除数据库中的文件记录。
//  3. 查询是否还有其他相同 digest 的文件记录。
//  4. 若无其他引用，则删除物理文件及对应的 .info 文件。
//
// 参数：
//   - ctx: 上下文。
//   - tx: 数据库事务，为 nil 时自动创建新事务。
//   - fileID: 要删除的文件记录主键ID。
//
// 返回值：
//   - err: 操作结果错误，成功时为 nil，失败时为具体错误信息。
//
// 补充：
//   - 文件删除为异步操作，不阻塞，也不影响事务回滚与返回值。
//   - 推荐在业务末端调用，避免回滚后物理文件已被删除。
func DeleteFileRecord(ctx context.Context, tx pgx.Tx, fileID int64) (err error) {

	var forceResult string
	if val := ctx.Value("DeleteFileRecord-force-result"); val != nil {
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

	// 拿到digest和文件存储路径
	var digest, filePath string
	err = tx.QueryRow(ctx, "SELECT digest, path FROM t_file WHERE id = $1", fileID).Scan(
		&digest, &filePath)
	if forceResult == "tx.QueryRow" {
		err = fmt.Errorf("强制查询文件信息错误")
	}
	if err != nil {
		z.Error(err.Error())
		return err
	}

	// 删除该文件记录
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
