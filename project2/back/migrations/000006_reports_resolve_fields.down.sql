alter table reports
	drop column if exists resolved_by,
	drop column if exists resolved_at;

