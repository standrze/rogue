/*
Copyright © 2025 Stephen Andrzejewski sandrzejewski@berkeley.edu
*/
package cmd

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/standrze/rogue/internal/config"
	"github.com/standrze/rogue/internal/proxy"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rogue",
	Short: "Rogue is a high-performance, configurable HTTP/HTTPS proxy server",
	Long: `Rogue is a versatile HTTP/HTTPS proxy server designed for deep traffic inspection and modification.
It features automatic certificate generation for MITM capabilities, comprehensive request/response logging,
and a flexible configuration system. Ideal for debugging, security testing, and traffic analysis.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Launch the Rogue proxy server instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set defaults
		defaultConfig := config.DefaultConfig()
		viper.SetDefault("proxy.port", defaultConfig.Proxy.Port)
		viper.SetDefault("proxy.host", defaultConfig.Proxy.Host)
		viper.SetDefault("proxy.timeout", defaultConfig.Proxy.Timeout)
		viper.SetDefault("certificate.auto_generate", defaultConfig.Certificate.AutoGenerate)
		viper.SetDefault("certificate.organization", defaultConfig.Certificate.Organization)
		viper.SetDefault("certificate.common_name", defaultConfig.Certificate.CommonName)
		viper.SetDefault("certificate.valid_days", defaultConfig.Certificate.ValidDays)
		viper.SetDefault("certificate.cert_path", defaultConfig.Certificate.CertPath)
		viper.SetDefault("certificate.key_path", defaultConfig.Certificate.KeyPath)
		viper.SetDefault("logging.session_dir", defaultConfig.Logging.SessionDir)
		viper.SetDefault("logging.log_requests", defaultConfig.Logging.LogRequests)
		viper.SetDefault("logging.log_responses", defaultConfig.Logging.LogResponses)
		viper.SetDefault("logging.log_headers", defaultConfig.Logging.LogHeaders)
		viper.SetDefault("logging.log_body", defaultConfig.Logging.LogBody)
		viper.SetDefault("logging.max_body_size", defaultConfig.Logging.MaxBodySize)

		viper.SetConfigName("config")
		viper.SetConfigType("json")
		viper.AddConfigPath(".")
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return err
			}
			// Config file not found; ignore error if desired or warn user
			fmt.Println("No config file found, using defaults and flags")
		}

		var cfg config.Config
		if err := viper.Unmarshal(&cfg); err != nil {
			return err
		}

		fmt.Printf("Starting Rogue on %s:%d\n", cfg.Proxy.Host, cfg.Proxy.Port)

		p := proxy.NewProxyServer(
			proxy.WithPort(cfg.Proxy.Port),
			proxy.WithHost(cfg.Proxy.Host),
			proxy.WithCert(cfg.Certificate.CertPath, cfg.Certificate.KeyPath),
			proxy.WithSessionDir(cfg.Logging.SessionDir),
			proxy.WithLogging(
				cfg.Logging.LogRequests,
				cfg.Logging.LogResponses,
				cfg.Logging.LogHeaders,
				cfg.Logging.LogBody,
				cfg.Logging.MaxBodySize,
			),
		)

		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Proxy.Host, cfg.Proxy.Port))
		if err != nil {
			return err
		}

		return p.Serve(l)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.ma=in(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.AddCommand(startCmd)
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rogue.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().IntP("port", "p", 8080, "Port for proxy server")
	viper.BindPFlag("proxy.port", rootCmd.PersistentFlags().Lookup("port"))
}
