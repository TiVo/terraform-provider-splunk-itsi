package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tivo/terraform-provider-splunk-itsi/itsictl/config"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider"
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

const (
	ITSICTL = "itsictl"
)

var (
	cfgFile string
	cfg     config.Config
)

var (
	services []string
	kpis     []string
)

var rootCmd = &cobra.Command{
	Use:   ITSICTL,
	Short: "A tool to manage Splunk ITSI",
	Long:  "A tool to manage Splunk ITSI",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
	},
}

func initConfig() {
	if cfgFile == "" {
		cfgFileEnv := os.Getenv("ITSICTL_CONFIG")
		if cfgFileEnv == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Println("Error finding home directory:", err)
				os.Exit(1)
			}

			// Search for config file in home directory with name ".itsictl"
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".itsictl")
		} else {
			viper.SetConfigFile(cfgFileEnv)
		}
	} else {
		// Use specified config file
		viper.SetConfigFile(cfgFile)
	}

	// Read in environment variables that match
	viper.SetEnvPrefix(ITSICTL)
	viper.AutomaticEnv()

	// Read in config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error has occurred
			fmt.Println("Error reading config file:", err)
			os.Exit(1)
		}
		// Config file not found; ignore
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Println("Error unmarshaling config:", err)
		os.Exit(1)
	}

	authentication := fmt.Sprintf("basic (%s)", cfg.User)

	switch {
	case cfg.AccessToken != "":
		authentication = "bearer token"
	case cfg.User != "":
		if cfg.Password == "" {
			fmt.Print("Enter password: ")
			pwd, err := util.ReadPassword()
			if err != nil {
				fmt.Println("\nError reading password:", err)
				os.Exit(1)
			}
			cfg.Password = pwd
		}
		fallthrough
	default:
		if cfg.User == "" || cfg.Password == "" {
			fmt.Println("Must provider user/password or access token")
			os.Exit(1)
		}
	}

	if cfg.Verbose {
		fmt.Println("----------------------------------------------------------")
		fmt.Println("CONFIG:")
		fmt.Printf("\tHost: %s\n", cfg.Host)
		fmt.Printf("\tPort: %d\n", cfg.Port)
		fmt.Printf("\tInsecure: %v\n", cfg.Insecure)
		fmt.Printf("\tConcurrency: %d\n", cfg.Concurrency)
		fmt.Printf("\tAuthentication: %s\n", authentication)
		fmt.Println("----------------------------------------------------------")
	}

	provider.InitSplunkSearchLimiter(cfg.Concurrency)
	models.InitItsiApiLimiter(cfg.Concurrency)
}

func init() {
	// General Options
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default is $HOME/.itsictl.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Int("concurrency", 10, "Number of concurrent operations")

	// Connection Options
	rootCmd.PersistentFlags().String("host", "localhost", "ITSI host")
	rootCmd.PersistentFlags().Int("port", 8089, "ITSI port")
	rootCmd.PersistentFlags().Bool("insecure", false, "Disable TLS certificate verification")

	// Authentication Options
	rootCmd.PersistentFlags().String("access-token", "", "Access token for authentication")
	rootCmd.PersistentFlags().String("user", "admin", "Username for authentication")
	rootCmd.PersistentFlags().String("password", "", "Password for authentication")

	// Bind flags to Viper

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("concurrency", rootCmd.PersistentFlags().Lookup("concurrency"))

	viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))

	viper.BindPFlag("access_token", rootCmd.PersistentFlags().Lookup("access-token"))
	viper.BindPFlag("user", rootCmd.PersistentFlags().Lookup("user"))
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
