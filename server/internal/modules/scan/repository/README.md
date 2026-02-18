# scan/repository

Repository conventions for the scan module:

- `scan.go` + `scan_query.go` + `scan_command.go`: three-way split for scan repository operations.
- `scan_mapper.go`: mapping between `persistence model <-> domain projection` (`domain.QueryScan/QueryTargetRef/QueryStatistics`) and create write models.
- `scan_domain_repository_adapter.go`: domain port adapter (`domain <-> repository`).
- `scan_log.go` + `scan_log_query.go` + `scan_log_command.go`: three-way split for log repository operations.
- `scan_task.go` + `scan_task_query.go` + `scan_task_command.go` + `scan_task_sql.go`: task repository and SQL constants.

Constraints:

- Do not use `*_mutation.go` naming.
- Do not use generic filenames like `types.go`.
- `*_query.go` must not contain write methods; `*_command.go` must not contain query methods.
- Keep query/projection mapping in repository; wiring should only adapt interfaces, not move fields.
