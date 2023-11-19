package bot

type Config struct {
	DiscordToken         string
	DiscordNotifyChannel string `mapstructure:"notify_channel"`
	Dibs                 []Dib  `mapstructure:"dibs"`
}

type DibType string

const (
	DibHTML DibType = "html"
	DibDiff DibType = "diff"
)

type Dib struct {
	Name      string  `mapstructure:"name"`
	Type      DibType `mapstructure:"type"`
	URL       string  `mapstructure:"url"`
	Selector  string  `mapstructure:"selector"`
	Condition string  `mapstructure:"if"`
	Message   string  `mapstructure:"message"`
	Interval  string  `mapstructure:"interval"`
}
