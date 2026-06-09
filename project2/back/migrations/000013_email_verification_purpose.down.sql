drop index if exists idx_email_verifications_email_purpose_created_at;

alter table email_verifications
drop column if exists purpose;
