package config

// Config holds all runtime configuration derived from CLI flags.
type Config struct {
	Namespace     string
	AllNamespaces bool
	Field         string
	LogLevel      string
	Kubeconfig    string
}
