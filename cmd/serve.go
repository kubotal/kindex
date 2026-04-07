package cmd

import (
	"context"
	"fmt"
	"kindex/internal/handlers"
	"kindex/internal/httpsrv"
	"kindex/internal/misc"
	"net/http"
	"os"
	"path"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

var flags struct {
	logConfig  misc.LogConfig
	httpConfig httpsrv.Config
}

func init() {
	ServeCmd.PersistentFlags().StringVarP(&flags.logConfig.Mode, "logMode", "", "text", "Log mode ('text' or 'json')")
	ServeCmd.PersistentFlags().StringVarP(&flags.logConfig.Level, "logLevel", "l", "INFO", "Log level(DEBUG, INFO, WARN, ERROR)")

	ServeCmd.PersistentFlags().BoolVarP(&flags.httpConfig.Tls, "tls", "t", false, "enable TLS")
	ServeCmd.PersistentFlags().IntVar(&flags.httpConfig.DumpExchanges, "dumpExchanges", 0, "Dump http server req/resp (0, 1, 2 or 3")
	ServeCmd.PersistentFlags().StringVarP(&flags.httpConfig.BindAddr, "bindAddr", "a", "0.0.0.0", "Bind Address")
	ServeCmd.PersistentFlags().IntVarP(&flags.httpConfig.BindPort, "bindPort", "p", 7777, "Bind port")
	ServeCmd.PersistentFlags().StringVar(&flags.httpConfig.CertDir, "certDir", "", "Certificate Directory")
	ServeCmd.PersistentFlags().StringVar(&flags.httpConfig.CertName, "certName", "tls.crt", "Certificate Directory")
	ServeCmd.PersistentFlags().StringVar(&flags.httpConfig.KeyName, "keyName", "tls.key", "Certificate Directory")
	//ServeCmd.PersistentFlags().StringArrayVarP(&flags.oidcHttpConfig.AllowedOrigins, "allowedOrigins", "", []string{}, "Allowed Origins")
}

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serves a http server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello World")

		logger, err := misc.NewLogger(&flags.logConfig)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Unable to load logging configuration: %v\n", err)
			os.Exit(2)
		}

		logger.Info("Starting http server", "port", flags.httpConfig.BindPort)

		router := http.NewServeMux()

		// Setup server
		server := httpsrv.New("ingresses", &flags.httpConfig, router)
		ctx := logr.NewContextWithSlogLogger(context.Background(), logger)

		router.Handle("GET /favicon.ico", handlers.FaviconHandler(path.Join("resources/static", "favicon.ico")))

		err = server.Start(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%error on server launchv\n", err)
			os.Exit(1)
		}

	},
}
