/*
Copyright 2025.

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

package main

import (
	"context"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"cluster/cmd"
	"cluster/internal/logx"
)

var mainLog = ctrl.Log.WithName("main")

func main() {
	ctx := ctrl.SetupSignalHandler()
	rootCmd, err := root()
	if err != nil {
		mainLog.Error(err, "unable to create cmd")
		os.Exit(1)
	}
	if err = rootCmd.ExecuteContext(ctx); err != nil {
		mainLog.Error(err, "problem running command")
		os.Exit(1)
	}
}

func initConfig(cmd *cobra.Command, _ []string) error {
	cfg, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	if cfg != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfg)
	} else {
		// Find home directory.
		var home string
		if home, err = homedir.Dir(); err != nil {
			return err
		}
		// Search config in home directory with name ".woden_client" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".cloud_term")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err = viper.ReadInConfig(); err == nil {
		mainLog.Info("Using config file", viper.ConfigFileUsed())
	}
	return nil
}

func root() (*cobra.Command, error) {
	opts := zap.Options{
		Development: true,
	}
	cmd := &cobra.Command{
		Use:   "manager",
		Short: "manager for cluster",
		Long:  "A command line tool for cloud term operator cluster.",
		// SilenceErrors: true,
		// SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
			return runRoot(cmd.Context())
		},
	}
	// Here you will define your flags and configuration settings.

	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	pf := cmd.PersistentFlags()
	pf.StringP("config", "c", "", "config file (default is $HOME/.cloud_term.yaml)")
	pf.StringP("metrics-bind-address", "", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	if err := viper.BindPFlag("metrics-bind-address", pf.Lookup("metrics-bind-address")); err != nil {
		return nil, err
	}
	pf.StringP("health-probe-bind-address", "", ":8081", "The address the probe endpoint binds to.")
	if err := viper.BindPFlag("health-probe-bind-address", pf.Lookup("health-probe-bind-address")); err != nil {
		return nil, err
	}
	pf.BoolP("leader-elect", "", false, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	if err := viper.BindPFlag("leader-elect", pf.Lookup("leader-elect")); err != nil {
		return nil, err
	}
	pf.BoolP("metrics-secure", "", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	if err := viper.BindPFlag("metrics-secure", pf.Lookup("metrics-secure")); err != nil {
		return nil, err
	}
	pf.StringP("webhook-cert-path", "", "", "The directory that contains the webhook certificate.")
	if err := viper.BindPFlag("webhook-cert-path", pf.Lookup("webhook-cert-path")); err != nil {
		return nil, err
	}
	pf.StringP("webhook-cert-name", "", "tls.crt", "The name of the webhook certificate file.")
	if err := viper.BindPFlag("webhook-cert-name", pf.Lookup("webhook-cert-name")); err != nil {
		return nil, err
	}
	pf.StringP("webhook-cert-key", "", "tls.key", "The name of the webhook key file.")
	if err := viper.BindPFlag("webhook-cert-key", pf.Lookup("webhook-cert-key")); err != nil {
		return nil, err
	}
	pf.StringP("metrics-cert-path", "", "", "The directory that contains the metrics server certificate.")
	if err := viper.BindPFlag("metrics-cert-path", pf.Lookup("metrics-cert-path")); err != nil {
		return nil, err
	}
	pf.StringP("metrics-cert-name", "", "tls.crt", "The name of the metrics server certificate file.")
	if err := viper.BindPFlag("metrics-cert-name", pf.Lookup("metrics-cert-name")); err != nil {
		return nil, err
	}
	pf.StringP("metrics-cert-key", "", "tls.key", "The name of the metrics server key file.")
	if err := viper.BindPFlag("metrics-cert-key", pf.Lookup("metrics-cert-key")); err != nil {
		return nil, err
	}
	pf.BoolP("enable-http2", "", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	if err := viper.BindPFlag("enable-http2", pf.Lookup("enable-http2")); err != nil {
		return nil, err
	}
	pf.StringP("otel-endpoint", "", "", "The endpoint for the OpenTelemetry collector. If not set, no metrics will be exported.")
	if err := viper.BindPFlag("otel-endpoint", pf.Lookup("otel-endpoint")); err != nil {
		return nil, err
	}
	logx.BindFlags(&opts, pf)
	return cmd, nil
}

func runRoot(ctx context.Context) error {
	g := pool.New().WithContext(ctx).WithCancelOnError()
	g.Go(func(ctx context.Context) error {
		return cmd.Operator(ctx)
	})
	g.Go(func(ctx context.Context) error {
		return run(ctx)
	})
	return g.Wait()
}
