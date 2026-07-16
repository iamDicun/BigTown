package realtime

type RealtimeModule struct {
	provider *Provider
}

func NewRealtimeModule(jwtSecret string) *RealtimeModule {
	return &RealtimeModule{provider: NewProvider(jwtSecret)}
}
