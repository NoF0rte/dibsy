package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/NoF0rte/dibsy/internal/bot"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var dibsy *bot.Dibsy
var config *bot.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dibsy",
	Short: "Start the dibsy discord bot",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		dibsy, err = bot.New(config)
		if err != nil {
			return err
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := dibsy.Start()
		if err != nil {
			return err
		}

		defer dibsy.Close()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		<-stop

		log.Println("Graceful shutdown")
		return nil
	},
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

	config = &bot.Config{}
	err = viper.Unmarshal(config)
	if err != nil {
		fmt.Printf("unable to decode into config struct, %v", err)
	}

	godotenv.Load(".env")

	config.DiscordToken = os.Getenv("DISCORD_BOT_TOKEN")
}
