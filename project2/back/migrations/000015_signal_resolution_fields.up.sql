alter table signals
    add column if not exists resolution_kind varchar(32),
    add column if not exists resolution_comment text;
