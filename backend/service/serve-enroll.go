package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5"
	"w2w.io/cmn"
)

// serveEnroll 升级版API注册方法，支持AccessActions字段
// 一个运行时API可以拥有多个访问操作，每个访问操作作为独立的API记录插入到t_api表
func serveEnroll() (err error) {
	type api struct {
		ID           int64             `json:"id"`
		Name         string            `json:"name"`
		Exists       bool              `json:"-"`
		ExposePath   string            `json:"expose_path"`
		Author       *cmn.ModuleAuthor `json:"author"`
		AccessAction string            `json:"access_action"`
		Configurable bool              `json:"configurable"` // 新增可配置字段
	}
	ctx := context.Background()
	conn := cmn.GetPgxConn()
	s := `select json_agg(JSON_build_OBJECT(
		'id',id,
		'name',name,
		'expose_path',expose_path,
		'author',author,
		'access_action',coalesce(access_action, ''),
		'configurable',coalesce(configurable, false)
                )) as apis 
	from t_api`

	var apis []api
	err = conn.QueryRow(ctx, s).Scan(&apis)
	if err != nil {
		log.Fatal(err.Error())
	}

	var tx pgx.Tx
	tx, err = conn.BeginTx(ctx, pgx.TxOptions{})

	defer func() { _ = tx.Commit(ctx) }()
	var dstApis []api
	var newAuthor, oldAuthor []byte

	// 处理运行时API，构建待处理的API列表
	var runningApis []api
	for _, runningAPI := range cmn.Services {
		// 如果ApiEntries为空，使用原有逻辑，只插入一条API记录
		if len(runningAPI.ApiEntries) == 0 {
			runningApis = append(runningApis, api{
				Name:         runningAPI.Name,
				ExposePath:   runningAPI.Path,
				Author:       runningAPI.Developer,
				AccessAction: "",    // 空的访问操作
				Configurable: false, // 默认不可配置
			})
		} else {
			// 如果ApiEntries不为空，为每个访问操作创建一条API记录
			for _, accessAction := range runningAPI.ApiEntries {
				runningApis = append(runningApis, api{
					Name:         accessAction.Name,
					ExposePath:   runningAPI.Path,
					Author:       runningAPI.Developer,
					AccessAction: accessAction.AccessAction,
					Configurable: accessAction.Configurable,
				})
			}
		}
	}

	// 第一阶段：以name为主键进行匹配和更新
	for i, runningAPI := range runningApis {
		for _, existsAPI := range apis {
			if runningAPI.Name == existsAPI.Name {
				// 检查其他字段是否发生变化
				var authorChanged, exposePathChanged, accessActionChanged, configurableChanged bool

				// 检查Author字段变化
				if runningAPI.Author != nil && existsAPI.Author != nil {
					newAuthor, err = json.Marshal(runningAPI.Author)
					if err != nil {
						log.Fatal(err.Error())
					}
					oldAuthor, err = json.Marshal(existsAPI.Author)
					if err != nil {
						log.Fatal(err.Error())
					}
					authorChanged = string(newAuthor) != string(oldAuthor)
				} else if runningAPI.Author != existsAPI.Author {
					authorChanged = true
				}

				exposePathChanged = runningAPI.ExposePath != existsAPI.ExposePath
				accessActionChanged = runningAPI.AccessAction != existsAPI.AccessAction
				configurableChanged = runningAPI.Configurable != existsAPI.Configurable

				// 如果任何字段发生变化，标记为需要更新
				if authorChanged || exposePathChanged || accessActionChanged || configurableChanged {
					runningApis[i].ID = existsAPI.ID
					runningApis[i].Exists = true
					dstApis = append(dstApis, runningApis[i])
				} else {
					// 标记为已存在但不需要更新
					runningApis[i].Exists = true
				}
				break
			}
		}
	}

	// 第二阶段：以expose_path+access_action为组合键进行匹配和更新
	for i, runningAPI := range runningApis {
		// 跳过已经在第一阶段处理过的API
		if runningAPI.Exists {
			continue
		}

		for _, existsAPI := range apis {
			if runningAPI.ExposePath == existsAPI.ExposePath && runningAPI.AccessAction == existsAPI.AccessAction {
				// 检查其他字段是否发生变化
				var nameChanged, authorChanged, configurableChanged bool

				nameChanged = runningAPI.Name != existsAPI.Name

				// 检查Author字段变化
				if runningAPI.Author != nil && existsAPI.Author != nil {
					newAuthor, err = json.Marshal(runningAPI.Author)
					if err != nil {
						log.Fatal(err.Error())
					}
					oldAuthor, err = json.Marshal(existsAPI.Author)
					if err != nil {
						log.Fatal(err.Error())
					}
					authorChanged = string(newAuthor) != string(oldAuthor)
				} else if runningAPI.Author != existsAPI.Author {
					authorChanged = true
				}

				configurableChanged = runningAPI.Configurable != existsAPI.Configurable

				// 如果任何字段发生变化，标记为需要更新
				if nameChanged || authorChanged || configurableChanged {
					runningApis[i].ID = existsAPI.ID
					runningApis[i].Exists = true
					dstApis = append(dstApis, runningApis[i])
				} else {
					// 标记为已存在但不需要更新
					runningApis[i].Exists = true
				}
				break
			}
		}
	}

	// 添加新的API（既没有通过name匹配，也没有通过expose_path+access_action匹配的）
	for _, runningAPI := range runningApis {
		if !runningAPI.Exists {
			dstApis = append(dstApis, runningAPI)
		}
	}

	for _, v := range dstApis {
		var rtnID int64
		if v.Exists {
			s := `update t_api set name=$1,author=$2,access_action=$3,configurable=$4 where id=$5 returning id`
			var buf []byte
			buf, err = json.Marshal(v.Author)
			if err != nil {
				log.Fatal(err.Error())
			}
			err = tx.QueryRow(context.Background(), s, v.Name, string(buf), v.AccessAction, v.Configurable, v.ID).Scan(&rtnID)
			if err != nil {
				_ = tx.Rollback(ctx)
				log.Fatal(err.Error())
			}
			if rtnID != v.ID {
				log.Fatal("api id does not match")
			}
			continue
		}

		s = `insert into t_api(name,expose_path,author,access_control_level,
      maintainer,creator,domain_id,updated_by,access_action,configurable) 
			values 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			returning id`
		err = tx.QueryRow(context.Background(), s,
			v.Name, v.ExposePath, v.Author, "0", 1000, 1000, 322, 1000, v.AccessAction, v.Configurable).Scan(&rtnID)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Fatal(err.Error())
		}
		if rtnID == 0 {
			log.Fatal("api id does not match")
		}
	}

	return
}
