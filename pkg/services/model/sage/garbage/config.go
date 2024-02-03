package garbage

type Config struct {
	EnableGarbageCollection   bool   `yaml:"enable_garbage_collection,omitempty"`
	RemoveAfterDaysUnused     int    `yaml:"remove_endpoints_after_days_unused,omitempty"`
	CollectionWaitTimeMinutes int    `yaml:"collection_cycle_wait_time_minutes,omitempty"`
	ResourceEnv               string `yaml:"resource_env"`
}
