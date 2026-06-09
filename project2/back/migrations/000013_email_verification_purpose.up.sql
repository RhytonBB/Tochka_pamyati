alter table email_verifications
add column if not exists purpose varchar(50) not null default 'verify_email';

create index if not exists idx_email_verifications_email_purpose_created_at
on email_verifications (email, purpose, created_at desc);
