drop index if exists idx_comments_parent;

alter table comments
	drop column if exists parent_id;
