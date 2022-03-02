package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"text/template"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"

	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type dibbedElement struct {
	Attr map[string]string
	Text string
}

type Dibsy struct {
	discord *discordgo.Session
	cron    *cron.Cron
}

func (d Dibsy) ScheduleDib(dib Dib) error {
	_, err := d.cron.AddFunc(fmt.Sprintf(`@every %s`, dib.Interval), func() {
		log.Printf("Executing dib: %s\n", dib.Name)
		d.ExecDib(dib)
	})
	return err
}

func (d Dibsy) ExecDib(dib Dib) {
	collector := colly.NewCollector()
	collector.OnHTML(dib.Selector, func(h *colly.HTMLElement) {
		element := dibbedElement{
			Attr: make(map[string]string),
			Text: h.Text,
		}
		for _, node := range h.DOM.Nodes {
			for _, attr := range node.Attr {
				element.Attr[attr.Key] = attr.Val
			}
		}

		funcMap := make(template.FuncMap)
		funcMap["ieq"] = func(s1, s2 string) bool {
			return strings.EqualFold(s1, s2)
		}

		t, err := template.New("dib").Funcs(funcMap).Parse(fmt.Sprintf(`{{if %s}}true{{end}}`, dib.Condition))
		if err != nil {
			log.Println(err)
			return
		}

		var buf bytes.Buffer
		err = t.Execute(&buf, element)
		if err != nil {
			log.Println(err)
			return
		}

		if buf.String() == "" {
			return
		}

		message := fmt.Sprintf("New Dib!\n%s\n\n%s", dib.Message, dib.Url)
		_, err = d.discord.ChannelMessageSend(config.DiscordNotifyChannel, message)
		if err != nil {
			log.Println(err)
		}
	})
	collector.Visit(dib.Url)
}

func (d Dibsy) Start() error {
	log.Println("Starting dibsy...")
	err := d.discord.Open()
	if err != nil {
		return err
	}
	d.cron.Start()
	return nil
}

func (d Dibsy) Close() {
	log.Println("Stopping dibsy...")
	d.discord.Close()
	ctx := d.cron.Stop()
	ctx.Done()
}

var dibsy Dibsy
var config *DibsyConfig

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dibsy",
	Short: "start the dibsy discord bot",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		discord, err := discordgo.New("Bot " + config.DiscordToken)
		if err != nil {
			return err
		}
		dibsy = Dibsy{
			discord: discord,
			cron:    cron.New(),
		}

		for _, dib := range config.Dibs {
			err = dibsy.ScheduleDib(dib)
			if err != nil {
				return err
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		dibsy.Start()
		defer dibsy.Close()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		<-stop

		log.Println("Graceful shutdown")
		return nil
	},
}

type DibsyConfig struct {
	DiscordToken         string
	DiscordNotifyChannel string `mapstructure:"notify_channel"`
	Dibs                 []Dib  `mapstructure:"dibs"`
}

type Dib struct {
	Name      string `mapstructure:"name"`
	Url       string `mapstructure:"url"`
	Selector  string `mapstructure:"selector"`
	Condition string `mapstructure:"if"`
	Message   string `mapstructure:"message"`
	Interval  string `mapstructure:"interval"`
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigName("dibsy")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("%v", err)
	}

	config = &DibsyConfig{}
	err = viper.Unmarshal(config)
	if err != nil {
		fmt.Printf("unable to decode into config struct, %v", err)
	}

	godotenv.Load(".env")

	config.DiscordToken = os.Getenv("DISCORD_BOT_TOKEN")
}
