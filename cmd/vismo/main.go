package main

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/Veinar/vismo/internal/config"
	"github.com/Veinar/vismo/internal/k8s"
	"github.com/Veinar/vismo/internal/managedfields"
	"github.com/Veinar/vismo/internal/output"
	"github.com/Veinar/vismo/internal/source"
)

// Version is injected at build time via -ldflags "-X main.Version=<tag>".
var Version = "dev"

//go:embed banner.txt
var bannerText string

func main() {
	cfg := &config.Config{}

	root := &cobra.Command{
		Use:          "vismo",
		Short:        "Kubernetes configuration origin tracer",
		Long:         bannerText + "\nvismo answers: \"Why does this configuration look this way?\"",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initLogging(cfg.LogLevel)
		},
	}

	root.PersistentFlags().StringVarP(&cfg.Namespace, "namespace", "n", "", "Kubernetes namespace")
	root.PersistentFlags().BoolVarP(&cfg.AllNamespaces, "all-namespaces", "A", false, "All namespaces")
	root.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "warn", "Log level (debug, info, warn, error)")
	root.PersistentFlags().StringVar(&cfg.Kubeconfig, "kubeconfig", "", "Path to kubeconfig (default: $KUBECONFIG or ~/.kube/config)")

	root.AddCommand(newGetCmd(cfg))
	root.AddCommand(newVersionCmd())

	ctx, stop := signal.NotifyContext(root.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print vismo version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("vismo %s\n", Version)
		},
	}
}

func newGetCmd(cfg *config.Config) *cobra.Command {
	var fieldPath string

	cmd := &cobra.Command{
		Use:   "get <resource>[/<name>]",
		Short: "Get a Kubernetes resource and trace its configuration origin",
		Example: `  vismo get deploy/api -n production
  vismo get deployment/api -n production --field spec.replicas
  vismo get pods -n kube-system`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := k8s.NewClient(cfg.Kubeconfig)
			if err != nil {
				return fmt.Errorf("connect to cluster: %w", err)
			}

			resourceType, name := k8s.ParseResourceArg(args[0])
			gvr, err := k8s.ResolveGVR(ctx, client.Mapper(), resourceType)
			if err != nil {
				return err
			}

			ns := cfg.Namespace
			if cfg.AllNamespaces {
				ns = ""
			}

			formatter := output.NewTextFormatter(os.Stdout)

			// Single resource
			if name != "" {
				obj, err := client.GetResource(ctx, gvr, name, ns)
				if err != nil {
					return err
				}

				raw, err := yaml.Marshal(obj.Object)
				if err != nil {
					return fmt.Errorf("marshal YAML: %w", err)
				}
				formatter.RenderResource(raw)

				result := source.Analyze(obj)
				formatter.RenderSources(result)

				if fieldPath != "" {
					trace, err := managedfields.ParseOwners(obj, fieldPath)
					if err != nil {
						return fmt.Errorf("analyze managedFields: %w", err)
					}
					formatter.RenderFieldTrace(trace)
				}
				return nil
			}

			// List resources
			list, err := client.ListResources(ctx, gvr, ns)
			if err != nil {
				return err
			}
			for _, item := range list.Items {
				itemNs := item.GetNamespace()
				if itemNs != "" {
					fmt.Printf("%s/%s\n", itemNs, item.GetName())
				} else {
					fmt.Println(item.GetName())
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&fieldPath, "field", "", "Field path to trace ownership (e.g. spec.replicas)")
	return cmd
}

func initLogging(level string) error {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn", "":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		return fmt.Errorf("unknown log level %q", level)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})))
	return nil
}
