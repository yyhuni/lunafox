package repository

const failTasksSQL = `
	UPDATE scan_task st
	SET
		status = CASE
			WHEN s.deleted_at IS NOT NULL OR s.status NOT IN ('` + scanStatusPending + `', '` + scanStatusRunning + `') THEN '` + taskStatusCancelled + `'
			ELSE '` + taskStatusFailed + `'
		END,
		agent_id = NULL,
		completed_at = NOW(),
		error_message = CASE
			WHEN s.deleted_at IS NOT NULL THEN 'Scan deleted'
			WHEN s.status NOT IN ('` + scanStatusPending + `', '` + scanStatusRunning + `') THEN 'Scan already ended'
			ELSE 'Agent offline'
		END
	FROM scan s
	WHERE st.scan_id = s.id
		AND st.status = '` + taskStatusRunning + `'
		AND st.agent_id = ?
`

const pullTaskSQL = `
	WITH selected AS (
		SELECT st.id FROM scan_task st
		JOIN scan s ON s.id = st.scan_id
		WHERE st.status = '` + taskStatusPending + `'
		  AND s.status IN ('` + scanStatusPending + `', '` + scanStatusRunning + `')
		  AND s.deleted_at IS NULL
		ORDER BY st.stage DESC, st.created_at ASC
		LIMIT 1
		FOR UPDATE OF st SKIP LOCKED
	)
	UPDATE scan_task t
	SET status = '` + taskStatusRunning + `',
		agent_id = ?,
		started_at = NOW()
	FROM selected
	WHERE t.id = selected.id
	RETURNING t.*
`

const unlockNextStageSQL = `
	WITH next_stage AS (
		SELECT MIN(stage) AS stage
		FROM scan_task
		WHERE scan_id = ?
		  AND status = '` + taskStatusBlocked + `'
		  AND stage > ?
	)
	UPDATE scan_task
	SET status = '` + taskStatusPending + `'
	WHERE scan_id = ?
	  AND status = '` + taskStatusBlocked + `'
	  AND stage = (SELECT stage FROM next_stage)
`
