drop table if exists comment_ai_incidents;
drop table if exists user_sanctions;

drop index if exists idx_comments_deleted_at;
drop index if exists idx_comments_author;

alter table comments
	drop column if exists deleted_reason,
	drop column if exists deleted_by,
	drop column if exists deleted_at,
	drop column if exists edited_at;
