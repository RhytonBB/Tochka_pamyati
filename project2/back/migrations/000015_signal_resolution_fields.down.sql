alter table signals
    drop column if exists resolution_comment,
    drop column if exists resolution_kind;
