package marshal

type Config struct {
	Binary bool
	JSON   bool
}

type Option func(*Config)

func Binary() Option {
	return func(c *Config) {
		c.Binary = true
	}
}

func JSON() Option {
	return func(c *Config) {
		c.JSON = true
	}
}
