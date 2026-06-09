create extension if not exists pg_trgm;

alter table monuments
	add column if not exists high_risk boolean not null default false,
	add column if not exists ai_flags jsonb not null default '{}'::jsonb;

alter table posts
	add column if not exists high_risk boolean not null default false,
	add column if not exists ai_flags jsonb not null default '{}'::jsonb;

alter table photos
	add column if not exists ai_flags jsonb not null default '{}'::jsonb;

