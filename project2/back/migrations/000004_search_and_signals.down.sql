drop table if exists signal_supports;
drop table if exists signal_photos;

drop index if exists idx_posts_search_tsv;
alter table posts drop column if exists search_tsv;

drop index if exists idx_monuments_search_tsv;
alter table monuments drop column if exists search_tsv;

