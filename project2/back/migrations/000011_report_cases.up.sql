create table if not exists report_cases (
	id uuid primary key default gen_random_uuid(),
	entity_type varchar(50) not null,
	entity_id uuid not null,
	reason_code varchar(50) not null,
	category varchar(20) not null,
	severity varchar(20) not null default 'medium',
	reports_count integer not null default 0,
	distinct_reporters_count integer not null default 0,
	status varchar(50) not null default 'pending',
	entity_snapshot jsonb not null default '{}'::jsonb,
	suggested_fix jsonb not null default '{}'::jsonb,
	priority_score integer not null default 0,
	last_reported_at timestamptz not null default now(),
	resolved_at timestamptz,
	resolved_by uuid references users(id) on delete set null,
	resolution_action varchar(50),
	moderator_comment text,
	created_at timestamptz not null default now()
);

create unique index if not exists ux_report_cases_open
	on report_cases (entity_type, entity_id, reason_code)
	where status in ('pending', 'auto_hidden');

create index if not exists idx_report_cases_status on report_cases (status);
create index if not exists idx_report_cases_priority on report_cases (priority_score desc, last_reported_at desc);

create table if not exists report_votes (
	id uuid primary key default gen_random_uuid(),
	case_id uuid not null references report_cases(id) on delete cascade,
	entity_type varchar(50) not null,
	entity_id uuid not null,
	reporter_id uuid not null references users(id) on delete cascade,
	reason_code varchar(50) not null,
	comment text,
	status varchar(50) not null default 'pending',
	entity_snapshot jsonb not null default '{}'::jsonb,
	suggested_fix jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	resolved_at timestamptz,
	resolved_by uuid references users(id) on delete set null
);

create unique index if not exists ux_report_votes_active
	on report_votes (reporter_id, entity_type, entity_id, reason_code)
	where status = 'pending';

create index if not exists idx_report_votes_case_id on report_votes (case_id);
