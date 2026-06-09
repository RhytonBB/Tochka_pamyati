create extension if not exists postgis;
create extension if not exists pgcrypto;

create table if not exists roles (
	id uuid primary key default gen_random_uuid(),
	name varchar(50) not null unique,
	permissions jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create table if not exists users (
	id uuid primary key default gen_random_uuid(),
	username varchar(100) not null unique,
	email varchar(255) not null unique,
	password_hash varchar(255) not null,
	role_id uuid not null references roles(id),
	trust_score integer not null default 5,
	city varchar(100),
	notification_settings jsonb not null default '{}'::jsonb,
	is_active boolean not null default false,
	is_blocked boolean not null default false,
	created_at timestamptz not null default now(),
	last_login timestamptz
);

create table if not exists email_verifications (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references users(id) on delete cascade,
	email varchar(255) not null,
	code varchar(6) not null,
	expires_at timestamptz not null,
	created_at timestamptz not null default now(),
	used_at timestamptz,
	attempts integer not null default 0
);

create index if not exists idx_email_verifications_email_created_at on email_verifications (email, created_at desc);

create table if not exists user_sessions (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references users(id) on delete cascade,
	refresh_jti uuid not null unique,
	expires_at timestamptz not null,
	created_at timestamptz not null default now(),
	revoked_at timestamptz,
	ip text,
	user_agent text
);

create table if not exists monuments (
	id uuid primary key default gen_random_uuid(),
	name varchar(255) not null,
	geom geometry(Point, 4326) not null,
	properties jsonb not null default '{}'::jsonb,
	status varchar(50) not null default 'pending',
	author_id uuid references users(id) on delete set null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	moderation_comment text
);

create index if not exists idx_monuments_geom on monuments using gist (geom);
create index if not exists idx_monuments_status on monuments (status);
create index if not exists idx_monuments_author on monuments (author_id);

create table if not exists posts (
	id uuid primary key default gen_random_uuid(),
	monument_id uuid not null references monuments(id) on delete cascade,
	author_id uuid not null references users(id) on delete cascade,
	description text,
	status varchar(50) not null default 'pending',
	edited_at timestamptz,
	created_at timestamptz not null default now(),
	moderation_comment text,
	toxic_score double precision
);

create index if not exists idx_posts_monument on posts (monument_id);
create index if not exists idx_posts_author on posts (author_id);
create index if not exists idx_posts_status on posts (status);

create table if not exists photos (
	id uuid primary key default gen_random_uuid(),
	post_id uuid not null references posts(id) on delete cascade,
	file_path varchar(500),
	thumbnail_path varchar(500),
	preview_path varchar(500),
	exif_data jsonb not null default '{}'::jsonb,
	relevance_score double precision,
	uploaded_at timestamptz not null default now()
);

create index if not exists idx_photos_post on photos (post_id);

create table if not exists monument_relations (
	id uuid primary key default gen_random_uuid(),
	parent_id uuid not null references monuments(id) on delete cascade,
	child_id uuid not null references monuments(id) on delete cascade,
	relation_type varchar(50) not null,
	created_at timestamptz not null default now(),
	unique (parent_id, child_id, relation_type)
);

create table if not exists signals (
	id uuid primary key default gen_random_uuid(),
	monument_id uuid references monuments(id) on delete set null,
	monument_name varchar(255),
	monument_location geometry(Point, 4326),
	signal_type varchar(50) not null,
	urgency varchar(20) not null default 'medium',
	description text not null,
	author_id uuid references users(id) on delete cascade,
	status varchar(50) not null default 'pending',
	official_response text,
	created_at timestamptz not null default now(),
	resolved_at timestamptz
);

create index if not exists idx_signals_status on signals (status);
create index if not exists idx_signals_geom on signals using gist (monument_location);

create table if not exists comments (
	id uuid primary key default gen_random_uuid(),
	signal_id uuid not null references signals(id) on delete cascade,
	author_id uuid references users(id) on delete cascade,
	content text not null,
	is_hidden boolean not null default false,
	toxic_score double precision,
	created_at timestamptz not null default now()
);

create index if not exists idx_comments_signal on comments (signal_id);

create table if not exists reports (
	id uuid primary key default gen_random_uuid(),
	entity_type varchar(50) not null,
	entity_id uuid not null,
	reporter_id uuid not null references users(id) on delete cascade,
	report_type varchar(50) not null,
	comment text,
	status varchar(50) not null default 'pending',
	created_at timestamptz not null default now()
);

create index if not exists idx_reports_status on reports (status);

create table if not exists notifications (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references users(id) on delete cascade,
	type varchar(50) not null,
	title varchar(255) not null,
	content text not null,
	link varchar(500),
	is_read boolean not null default false,
	created_at timestamptz not null default now()
);

create index if not exists idx_notifications_user on notifications (user_id, is_read, created_at desc);

create table if not exists audit_log (
	id uuid primary key default gen_random_uuid(),
	entity_type varchar(50) not null,
	entity_id uuid not null,
	field_name varchar(100) not null,
	old_value text,
	new_value text,
	author_id uuid references users(id) on delete set null,
	status varchar(50) not null,
	created_at timestamptz not null default now(),
	moderated_at timestamptz,
	moderated_by uuid references users(id) on delete set null
);

