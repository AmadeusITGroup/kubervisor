package breaker

//FactoryConfig parameters required for the creation of a breaker
type FactoryConfig struct {
	Config
	customFactory Factory
}

//Factory func for Breaker
type Factory func(cfg FactoryConfig) (Breaker, error)

var _ Factory = New

//New Factory for AnomalyDetection
func New(cfg FactoryConfig) (Breaker, error) {
	if cfg.customFactory != nil {
		return cfg.customFactory(cfg)
	}

	return nil, nil
}
