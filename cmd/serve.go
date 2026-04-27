/*
Copyright (c) 2026 Kubotal.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"
	"kindex/internal/global"
	"kindex/internal/handlers"
	"kindex/internal/httpsrv"
	"kindex/internal/misc"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	gatewayversioned "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var flags struct {
	logConfig   misc.LogConfig
	httpConfig  httpsrv.Config
	kubeconfig  string
	mode        string
	clusterName string
}

func init() {
	ServeCmd.PersistentFlags().StringVarP(&flags.logConfig.Mode, "logMode", "", "text", "Log mode ('text' or 'json')")
	ServeCmd.PersistentFlags().StringVarP(&flags.logConfig.Level, "logLevel", "l", "INFO", "Log level(DEBUG, INFO, WARN, ERROR)")

	ServeCmd.PersistentFlags().BoolVarP(&flags.httpConfig.Tls, "tls", "t", false, "enable TLS")
	ServeCmd.PersistentFlags().IntVar(&flags.httpConfig.DumpExchanges, "dumpExchanges", 0, "Dump http server req/resp (0, 1, 2 or 3")
	ServeCmd.PersistentFlags().StringVarP(&flags.httpConfig.BindAddr, "bindAddr", "a", "0.0.0.0", "Bind Address")
	ServeCmd.PersistentFlags().IntVarP(&flags.httpConfig.BindPort, "bindPort", "p", 7788, "Bind port")
	ServeCmd.PersistentFlags().StringVar(&flags.httpConfig.CertDir, "certDir", "", "Certificate Directory")
	ServeCmd.PersistentFlags().StringVar(&flags.httpConfig.CertName, "certName", "tls.crt", "Certificate Directory")
	ServeCmd.PersistentFlags().StringVar(&flags.httpConfig.KeyName, "keyName", "tls.key", "Certificate Directory")
	ServeCmd.PersistentFlags().StringVarP(&flags.kubeconfig, "kubeconfig", "k", "", "Kubeconfig file (overrides $KUBECONFIG and ~/.kube/config)")
	//ServeCmd.PersistentFlags().StringArrayVarP(&flags.oidcHttpConfig.AllowedOrigins, "allowedOrigins", "", []string{}, "Allowed Origins")
	ServeCmd.PersistentFlags().StringVar(&flags.mode, "mode", "dark", "Display mode: dark or light")
	ServeCmd.PersistentFlags().StringVar(&flags.clusterName, "clusterName", "", "Cluster Name")

}

var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Runs an HTTP server that lists ingress links from the cluster",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logger, err := misc.NewLogger(&flags.logConfig)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Unable to load logging configuration: %v\n", err)
			os.Exit(2)
		}

		mode := strings.ToLower(strings.TrimSpace(flags.mode))
		if mode != "dark" && mode != "light" {
			_, _ = fmt.Fprintf(os.Stderr, "--mode must be 'dark' or 'light', got %q\n", flags.mode)
			os.Exit(2)
		}

		logger.Info("Starting kindex server", "port", flags.httpConfig.BindPort, "version", global.Version, "build", global.BuildTs, "logLevel", flags.logConfig.Level, "mode", mode)

		// Kubeconfig resolution (first match wins):
		// 1) --kubeconfig path
		// 2) else KUBECONFIG environment variable (standard merge of listed files)
		// 3) else ~/.kube/config
		restCfg, clusterName, err := kubeAccessFromFlags(flags.kubeconfig)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "kubeconfig: %v\n", err)
			os.Exit(2)
		}
		if flags.clusterName != "" {
			clusterName = flags.clusterName
		}
		clientSet, err := kubernetes.NewForConfig(restCfg)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "kubernetes client: %v\n", err)
			os.Exit(2)
		}

		var gwClient gatewayversioned.Interface
		gwClient, err = gatewayversioned.NewForConfig(restCfg)
		if err != nil {
			logger.Info("gateway-api client unavailable; HTTPRoute/TLSRoute links disabled", "error", err)
			gwClient = nil
		}

		router := http.NewServeMux()

		// Setup server
		server := httpsrv.New("ingresses", &flags.httpConfig, router)
		ctx := logr.NewContextWithSlogLogger(context.Background(), logger)

		router.Handle("GET /", handlers.IngressesHandler(clientSet, gwClient, clusterName, mode))
		router.Handle("GET /favicon.ico", handlers.FaviconHandler(path.Join("resources/static", "favicon.ico")))

		err = server.Start(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error on server launch: %v\n", err)
			os.Exit(1)
		}

	},
}

// kubeAccessFromFlags returns a REST config and a display name for the cluster from the
// current kubeconfig context (the cluster entry name referenced by that context).
func kubeAccessFromFlags(explicitPath string) (*rest.Config, string, error) {
	if explicitPath != "" {
		apiCfg, err := clientcmd.LoadFromFile(explicitPath)
		if err != nil {
			return nil, "", err
		}
		restCfg, err := clientcmd.BuildConfigFromFlags("", explicitPath)
		if err != nil {
			return nil, "", err
		}
		return restCfg, kubeClusterDisplayName(apiCfg), nil
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	clientCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	apiCfg, err := clientCfg.RawConfig()
	if err != nil {
		return nil, "", err
	}
	restCfg, err := clientCfg.ClientConfig()
	if err != nil {
		return nil, "", err
	}
	return restCfg, kubeClusterDisplayName(&apiCfg), nil
}

func kubeClusterDisplayName(cfg *clientcmdapi.Config) string {
	if cfg == nil {
		return ""
	}
	ctxName := cfg.CurrentContext
	if ctxName == "" {
		return ""
	}
	ctx, ok := cfg.Contexts[ctxName]
	if !ok {
		return ctxName
	}
	if ctx.Cluster != "" {
		return ctx.Cluster
	}
	return ctxName
}
