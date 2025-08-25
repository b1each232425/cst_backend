package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5"
	"w2w.io/cmn"
)

func serveEnroll() (err error) {
	type api struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Exists bool   `json:"-"`

		ExposePath string `json:"expose_path"`

		Author *cmn.ModuleAuthor `json:"author"`
	}
	ctx := context.Background()
	conn := cmn.GetPgxConn()
	s := `select json_agg(JSON_build_OBJECT(
		'id',id,
		'name',name,
		'expose_path',
		expose_path,'author',author)) as apis 
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
outerFor:
	for exposePath, runningAPI := range cmn.Services {
		if runningAPI.Developer != nil {
			newAuthor, err = json.Marshal(runningAPI.Developer)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		nAPI := api{Name: runningAPI.Name, ExposePath: runningAPI.Path, Author: runningAPI.Developer}

		for _, existsAPI := range apis {
			if existsAPI.Author != nil {
				oldAuthor, err = json.Marshal(existsAPI.Author)
				if err != nil {
					log.Fatal(err.Error())
				}
			}

			if runningAPI.Name == existsAPI.Name && existsAPI.ExposePath == exposePath {
				if runningAPI.Developer == nil {
					continue outerFor
				}

				if existsAPI.Author == nil || existsAPI.Author != nil && string(oldAuthor) != string(newAuthor) {
					nAPI.ID = existsAPI.ID
					nAPI.Exists = true
					dstApis = append(dstApis, nAPI)
				}

				continue outerFor
			}

			if existsAPI.ExposePath == exposePath {
				nAPI.Exists = true
				nAPI.ID = existsAPI.ID
				nAPI.Author = existsAPI.Author
			}

			if runningAPI.Name == existsAPI.Name {
				nAPI.Name = runningAPI.PkgName + "/" + runningAPI.Name
			}
		}
		dstApis = append(dstApis, nAPI)
	}

	for _, v := range dstApis {
		var rtnID int64
		if v.Exists {
			s := `update t_api set name=$1,author=$2 where id=$3 returning id`
			var buf []byte
			buf, err = json.Marshal(v.Author)
			if err != nil {
				log.Fatal(err.Error())
			}
			err = tx.QueryRow(context.Background(), s, v.Name, string(buf), v.ID).Scan(&rtnID)
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
      maintainer,creator,domain_id,updated_by) 
			values 
			($1, $2, $3, $4, $5, $6, $7,$8)
			returning id`
		err = tx.QueryRow(context.Background(), s,
			v.Name, v.ExposePath, v.Author, "0", 1000, 1000, 322, 1000).Scan(&rtnID)
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
