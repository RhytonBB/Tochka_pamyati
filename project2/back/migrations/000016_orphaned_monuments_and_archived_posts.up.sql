alter table monuments
    add column if not exists is_orphaned boolean not null default false,
    add column if not exists orphaned_at timestamptz,
    add column if not exists orphaned_by_user_id uuid references users(id) on delete set null;

alter table posts
    add column if not exists is_archived boolean not null default false,
    add column if not exists archive_reason text,
    add column if not exists restore_decision_status varchar(30) not null default 'none',
    add column if not exists archived_at timestamptz,
    add column if not exists restored_at timestamptz;

create table if not exists admin_event_logs (
    id uuid primary key default gen_random_uuid(),
    actor_user_id uuid references users(id) on delete set null,
    target_user_id uuid references users(id) on delete set null,
    entity_type varchar(50) not null,
    entity_id uuid,
    action varchar(80) not null,
    result varchar(30) not null default 'success',
    message text not null,
    meta jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create index if not exists idx_admin_event_logs_actor on admin_event_logs(actor_user_id, created_at desc);
create index if not exists idx_admin_event_logs_target on admin_event_logs(target_user_id, created_at desc);
create index if not exists idx_admin_event_logs_entity on admin_event_logs(entity_type, entity_id, created_at desc);
create index if not exists idx_admin_event_logs_action on admin_event_logs(action, created_at desc);
