alter table signal_photos
	add column if not exists is_hidden boolean not null default false;
