package config

type Config struct {
	LogLevel string `default:"debug"`

	CoSign CoSignConfig

	Address string `default:"0.0.0.0:8080"`
}

// CoSignConfig contains the credentials for info exposed through the webapi
type CoSignConfig struct {
	Name     string `required:"true"`
	Password string `required:"true"`
}
