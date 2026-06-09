create table if not exists trust_events (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references users(id) on delete cascade,
	delta integer not null,
	reason_code varchar(100) not null,
	source_type varchar(50) not null,
	source_id uuid,
	comment text,
	created_at timestamptz not null default now()
);

create index if not exists idx_trust_events_user_created_at on trust_events (user_id, created_at desc);
