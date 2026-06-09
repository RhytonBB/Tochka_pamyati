package service

func notificationSettingEnabled(settings map[string]any, key string) bool {
	if len(settings) == 0 {
		return true
	}
	raw, ok := settings[key]
	if !ok {
		return true
	}
	enabled, ok := raw.(bool)
	if !ok {
		return true
	}
	return enabled
}
