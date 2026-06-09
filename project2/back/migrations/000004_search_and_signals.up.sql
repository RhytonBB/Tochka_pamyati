alter table monuments
	add column if not exists search_tsv tsvector generated always as (to_tsvector('russian', coalesce(name, ''))) stored;

create index if not exists idx_monuments_search_tsv on monuments using gin (search_tsv);

alter table posts
	add column if not exists search_tsv tsvector generated always as (to_tsvector('russian', coalesce(description, ''))) stored;

create index if not exists idx_posts_search_tsv on posts using gin (search_tsv);

create table if not exists signal_photos (
	id uuid primary key default gen_random_uuid(),
	signal_id uuid not null references signals(id) on delete cascade,
	file_path varchar(500),
	thumbnail_path varchar(500),
	preview_path varchar(500),
	exif_data jsonb not null default '{}'::jsonb,
	relevance_score double precision,
	ai_flags jsonb not null default '{}'::jsonb,
	uploaded_at timestamptz not null default now()
);

create index if not exists idx_signal_photos_signal on signal_photos (signal_id);

create table if not exists signal_supports (
	signal_id uuid not null references signals(id) on delete cascade,
	user_id uuid not null references users(id) on delete cascade,
	created_at timestamptz not null default now(),
	primary key (signal_id, user_id)
);

