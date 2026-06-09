alter table reports
	add column if not exists resolved_at timestamptz,
	add column if not exists resolved_by uuid references users(id) on delete set null;

