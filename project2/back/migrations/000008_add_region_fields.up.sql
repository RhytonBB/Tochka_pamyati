-- Add region column to monuments, signals and users
alter table monuments add column if not exists region varchar(100);
alter table signals add column if not exists region varchar(100);
alter table users add column if not exists region varchar(100);

-- Migrate existing city values to region for users if needed
update users set region = city where region is null and city is not null;
