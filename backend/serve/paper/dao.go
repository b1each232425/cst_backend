package paper

//func ValidateExistingPapers(ctx context.Context, tx pgx.Tx, paperIDs []int64) ([]int64, error) {
//	if len(paperIDs) == 0 {
//		z.Error(ErrEmptyPaperIDs.Error())
//		return []int64{}, nil
//	}
//	const sqlString = `SELECT id FROM t_paper WHERE id = ANY($1) AND status = '00'`
//	rows, err := tx.Query(ctx, sqlString, paperIDs)
//	if err != nil {
//		if errors.Is(err, pgx.ErrNoRows) {
//			z.Error(ErrPaperNotFound.Error())
//			return []int64{}, ErrPaperNotFound
//		}
//		z.Error(err.Error())
//		return []int64{}, err
//	}
//	defer rows.Close()
//
//	var validIDs []int64
//	for rows.Next() {
//		var id int64
//		if err := rows.Scan(&id); err != nil {
//			z.Error(err.Error())
//			return []int64{}, err
//		}
//		validIDs = append(validIDs, id)
//	}
//	if err := rows.Err(); err != nil {
//		z.Error(err.Error())
//		return []int64{}, err
//	}
//	return validIDs, nil
//}

//// 检查试卷是否存在
//func paperExists(ctx context.Context, paperID int64) (bool, error) {
//	z.Info("---->" + cmn.FncName())
//
//	if paperID <= 0 {
//		err := fmt.Errorf("无效的试卷ID: %d", paperID)
//		z.Error(err.Error())
//		return false, err
//	}
//
//	conn := cmn.GetPgxConn()
//	var exists bool
//	err := conn.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM t_paper WHERE id=$1 AND status!= '02')", paperID).Scan(&exists)
//	if val, ok := ctx.Value("force-error").(string); ok && val == "paperExists-QueryRow-err" {
//		err = errors.New(val)
//	}
//	if err != nil {
//		z.Error(err.Error())
//		return false, err
//	}
//
//	return exists, nil
//}

//// ---------------------------------------共享试卷----------------------------------------------
//func getPaperShareInfo(ctx context.Context, tx *pgx.Tx, paperID int64, req GetSharedUserListRequest) ([]cmn.TVPaperShare, int64, error) {
//	if paperID <= 0 {
//		z.Error(ErrInvalidPaperID.Error())
//		return []cmn.TVPaperShare{}, 0, ErrInvalidPaperID
//	}
//	err := cmn.Validate(req)
//	if err != nil {
//		z.Error(err.Error())
//		return []cmn.TVPaperShare{}, 0, err
//	}
//	offset := (req.Page - 1) * req.PageSize
//
//	// 构建动态 where 条件
//	var whereClauses []string
//	var params []interface{}
//	paramCount := 1
//
//	whereClauses = append(whereClauses, fmt.Sprintf("paper_id = $%d", paramCount))
//	params = append(params, paperID)
//	paramCount++
//
//	if req.Filter != "" {
//		whereClauses = append(whereClauses, fmt.Sprintf("(official_name ILIKE $%d OR mobile_phone ILIKE $%d OR account ILIKE $%d)", paramCount, paramCount, paramCount))
//		params = append(params, "%"+req.Filter+"%")
//		paramCount++
//	}
//
//	var whereClause string
//	if len(whereClauses) > 0 {
//		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
//	}
//
//	// 1. 查询总数
//	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM v_paper_share %s", whereClause)
//	var totalCount int64
//	err = tx.QueryRowContext(ctx, countSQL, params...).Scan(&totalCount)
//	if err != nil {
//		z.Error("failed to count paper share list", zap.Error(err))
//		return []cmn.TVPaperShare{}, 0, err
//	}
//
//	// 2. 查询分页数据
//	listSQL := fmt.Sprintf(`
//		SELECT user_id, official_name, account, mobile_phone, shared_time
//		FROM v_paper_share
//		%s
//		ORDER BY shared_time DESC
//		LIMIT $%d OFFSET $%d`,
//		whereClause, paramCount, paramCount+1)
//	dataParams := append(params, req.PageSize, offset)
//	rows, err := tx.QueryContext(ctx, listSQL, dataParams...)
//	if err != nil {
//		z.Error("failed to query paper share list", zap.Error(err))
//		return []cmn.TVPaperShare{}, 0, err
//	}
//	defer rows.Close()
//	var shares []cmn.TVPaperShare
//	for rows.Next() {
//		var share cmn.TVPaperShare
//		err := rows.Scan(&share.UserID, &share.OfficialName, &share.Account, &share.MobilePhone, &share.SharedTime)
//		if err != nil {
//			z.Error("failed to scan paper share info", zap.Error(err))
//			return []cmn.TVPaperShare{}, 0, err
//		}
//		shares = append(shares, share)
//	}
//	if err := rows.Err(); err != nil {
//		z.Error("rows iteration error", zap.Error(err))
//		return []cmn.TVPaperShare{}, 0, err
//	}
//	return shares, totalCount, nil
//}
//
//func managePaperShareUsers(ctx context.Context, tx *pgx.Tx, req ManagePaperShareRequest, currentUserID int64) error {
//	// 参数校验
//	if req.PaperID <= 0 || currentUserID <= 0 {
//		return fmt.Errorf("invalid paper id or user id")
//	}
//	err := validateIDs(req.UserIDs)
//	if err != nil {
//		return err
//	}
//
//	now := time.Now().UnixMilli()
//
//	switch req.Action {
//	case "add":
//		valueStrings := make([]string, 0, len(req.UserIDs))
//		valueArgs := make([]interface{}, 0, len(req.UserIDs)*8)
//		for i, userID := range req.UserIDs {
//			base := i * 8
//			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
//				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8))
//			valueArgs = append(valueArgs,
//				PaperResourceShareType, // type
//				req.PaperID,            // resource_id
//				userID,                 // user_id
//				currentUserID,          // creator
//				now,                    // create_time
//				currentUserID,          // updated_by
//				now,                    // update_time
//				StatusNormal,           // status
//			)
//		}
//		query := fmt.Sprintf(`
//			INSERT INTO t_resource_share
//				(type, resource_id, user_id, creator, create_time, updated_by, update_time, status)
//			VALUES %s
//			ON CONFLICT (type, resource_id, user_id) DO UPDATE
//			SET
//				status = EXCLUDED.status,
//				updated_by = EXCLUDED.updated_by,
//				update_time = EXCLUDED.update_time
//		`, strings.Join(valueStrings, ","))
//		result, err := tx.ExecContext(ctx, query, valueArgs...)
//		if err != nil {
//			z.Error(fmt.Sprintf("failed to manage paper share users (add): %v", err))
//			return err
//		}
//		rowsAffected, err := result.RowsAffected()
//		if err != nil {
//			z.Error(err.Error())
//			return err
//		}
//		if rowsAffected == 0 {
//			err := fmt.Errorf("no users were shared (paperID: %d)", req.PaperID)
//			z.Error(err.Error())
//			return err
//		}
//
//	case "remove":
//		result, err := tx.ExecContext(ctx,
//			`UPDATE t_resource_share
//			 SET status = $1,
//			     updated_by = $2,
//			     update_time = $3
//			 WHERE type = $4
//			   AND resource_id = $5
//			   AND user_id = ANY($6)
//			   AND status = $7`,
//			StatusUnNormal,         // $1
//			currentUserID,          // $2
//			now,                    // $3
//			PaperResourceShareType, // $4
//			req.PaperID,            // $5
//			pq.Array(req.UserIDs),  // $6
//			StatusNormal,           // $7
//		)
//		if err != nil {
//			z.Error(err.Error())
//			return err
//		}
//		rowsAffected, err := result.RowsAffected()
//		if err != nil {
//			z.Error(err.Error())
//			return err
//		}
//		if rowsAffected != int64(len(req.UserIDs)) {
//			err := fmt.Errorf("only %d of %d users removed (paperID: %d)",
//				rowsAffected, len(req.UserIDs), req.PaperID)
//			z.Error(err.Error())
//			return err
//		}
//	default:
//		err := fmt.Errorf("invalid action")
//		z.Error(err.Error())
//		return err
//	}
//	return nil
//}
//
//func updatePaperShareStatus(ctx context.Context, tx *pgx.Tx, req UpdatePaperAccessModeRequest, currentUserID int64) error {
//	now := time.Now().UnixMilli()
//	sql := `UPDATE t_resource_share
//	SET status = $1,
//		updated_by = $2,
//		update_time = $3
//		WHERE type = $4
//		AND resource_id = $5
//		`
//	result, err := tx.ExecContext(ctx, sql, req.AccessMode, currentUserID, now, PaperResourceShareType, req.PaperID)
//	if err != nil {
//		z.Error(err.Error())
//		return err
//	}
//	rowsAffected, err := result.RowsAffected()
//	if err != nil {
//		z.Error(err.Error())
//		return err
//	}
//	if rowsAffected == 0 {
//		err := fmt.Errorf("no users were shared (paperID: %d)", req.PaperID)
//		z.Error(err.Error())
//		return err
//	}
//	return nil
//}
//
//func validateUserPermissions(ctx context.Context, tx *pgx.Tx, paperID, userID int64) (bool, error) {
//	if paperID <= 0 {
//		z.Error(ErrInvalidPaperID.Error())
//		return false, ErrInvalidPaperID
//	}
//	if userID <= 0 {
//		z.Error(ErrInvalidUserID.Error())
//		return false, ErrInvalidUserID
//	}
//	sqlString := `
//	SELECT EXISTS(
//	SELECT 1
//	FROM t_paper
//	WHERE id = $1 AND (
//		creator = $2
//		OR access_mode = '04'
//		OR (
//			access_mode = '02'
//			AND EXISTS(
//				SELECT 1 FROM t_paper_share WHERE paper_id = $1 AND user_id = $2 AND status = '00'
//			)
//		)
//	)
//	)
//	`
//	var result bool
//	err := tx.QueryRowContext(ctx, sqlString, paperID, userID).Scan(&result)
//	if err != nil {
//		z.Error(err.Error())
//		return false, err
//	}
//	return result, nil
//}
//
//func validateUserIsPaperCreator(ctx context.Context, tx *pgx.Tx, paperID, userID int64) (bool, error) {
//	if paperID <= 0 {
//		z.Error(ErrInvalidPaperID.Error())
//		return false, ErrInvalidPaperID
//
//	}
//	if userID <= 0 {
//		z.Error(ErrInvalidUserID.Error())
//		return false, ErrInvalidUserID
//	}
//	sql := `
//SELECT EXISTS(
//SELECT 1
//FROM t_paper WHERE id = $1 AND creator = $2)`
//	var result bool
//	err := tx.QueryRowContext(ctx, sql, paperID, userID).Scan(&result)
//	if err != nil {
//		z.Error(err.Error())
//		return false, err
//	}
//	return result, nil
//}
