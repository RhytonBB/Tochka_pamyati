alter table photos
	drop column if exists is_hidden;

alter table posts
	drop column if exists is_hidden;

