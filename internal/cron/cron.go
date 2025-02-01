package cron

type Cron struct {
	Expression string `mapstructure:"expression,omitempty"`
}
