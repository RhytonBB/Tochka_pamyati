alter table posts
	add column if not exists is_hidden boolean not null default false;

alter table photos
	add column if not exists is_hidden boolean not null default false;

