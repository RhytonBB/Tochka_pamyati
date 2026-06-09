alter table comments
	add column if not exists edited_at timestamptz,
	add column if not exists deleted_at timestamptz,
	add column if not exists deleted_by uuid references users(id) on delete set null,
	add column if not exists deleted_reason text;

create index if not exists idx_comments_deleted_at on comments (deleted_at);
create index if not exists idx_comments_author on comments (author_id);

create table if not exists user_sanctions (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references users(id) on delete cascade,
	kind varchar(30) not null,
	source varchar(30) not null,
	reason_code varchar(80) not null,
	reason_text text,
	scopes text[] not null default '{}'::text[],
	starts_at timestamptz not null default now(),
	ends_at timestamptz,
	status varchar(30) not null default 'active',
	created_by uuid references users(id) on delete set null,
	related_entity_type varchar(50),
	related_entity_id uuid,
	meta jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	revoked_at timestamptz,
	revoked_by uuid references users(id) on delete set null,
	revoked_reason text
);

create index if not exists idx_user_sanctions_user on user_sanctions (user_id, status, starts_at desc);
create index if not exists idx_user_sanctions_active on user_sanctions (status, ends_at);

create table if not exists comment_ai_incidents (
	id uuid primary key default gen_random_uuid(),
	comment_id uuid not null references comments(id) on delete cascade,
	user_id uuid not null references users(id) on delete cascade,
	signal_id uuid not null references signals(id) on delete cascade,
	content_snapshot text not null,
	toxic_score double precision,
	event_type varchar(50) not null default 'ai_hidden',
	meta jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create index if not exists idx_comment_ai_incidents_user_time on comment_ai_incidents (user_id, created_at desc);
create index if not exists idx_comment_ai_incidents_comment on comment_ai_incidents (comment_id);
