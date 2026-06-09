insert into roles (name, permissions)
values
	('guest', '{}'::jsonb),
	('user', '{
		"can_create_monument": true,
		"can_create_post": true,
		"can_create_signal": true,
		"can_moderate_posts": false,
		"can_moderate_signals": false,
		"can_export_data": false,
		"can_manage_users": false
	}'::jsonb),
	('moderator', '{
		"can_create_monument": true,
		"can_create_post": true,
		"can_create_signal": true,
		"can_moderate_posts": true,
		"can_moderate_signals": true,
		"can_export_data": false,
		"can_manage_users": false
	}'::jsonb),
	('admin', '{
		"can_create_monument": true,
		"can_create_post": true,
		"can_create_signal": true,
		"can_moderate_posts": true,
		"can_moderate_signals": true,
		"can_export_data": true,
		"can_manage_users": true
	}'::jsonb)
on conflict (name) do nothing;

