drop index if exists idx_admin_event_logs_action;
drop index if exists idx_admin_event_logs_entity;
drop index if exists idx_admin_event_logs_target;
drop index if exists idx_admin_event_logs_actor;

drop table if exists admin_event_logs;

alter table posts
    drop column if exists restored_at,
    drop column if exists archived_at,
    drop column if exists restore_decision_status,
    drop column if exists archive_reason,
    drop column if exists is_archived;

alter table monuments
    drop column if exists orphaned_by_user_id,
    drop column if exists orphaned_at,
    drop column if exists is_orphaned;
