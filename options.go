package marshal

type Config struct {
	Binary bool
	JSON   bool
}

type Option func(*Config)

func Binary() Options {
	return func(c *Config) {
		c.Binary = true
	}
}

func JSON() Options {
	return func(c *Config) {
		c.JSON = true
	}
}
