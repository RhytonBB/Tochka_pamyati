alter table signals
	add column if not exists high_risk boolean not null default false,
	add column if not exists ai_flags jsonb not null default '{}'::jsonb;

