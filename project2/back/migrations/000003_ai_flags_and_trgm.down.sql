alter table photos
	drop column if exists ai_flags;

alter table posts
	drop column if exists ai_flags,
	drop column if exists high_risk;

alter table monuments
	drop column if exists ai_flags,
	drop column if exists high_risk;

