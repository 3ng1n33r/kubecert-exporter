package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
)

type CertTarget struct {
	Name  string `yaml:"name"`
	Path  string `yaml:"path"`
	Group string `yaml:"group"`
}

type Config struct {
	Certs []CertTarget `yaml:"certs"`
}

var certs []CertTarget

var (
	certExpiry = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeadm_cert_expiry_seconds",
			Help: "Expiry time of kubeadm certificates",
		},
		[]string{"name", "group", "path"},
	)
)

func init() {
	prometheus.MustRegister(certExpiry)
}

func loadConfig(path string) ([]CertTarget, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return cfg.Certs, nil
}

func readCertExpiry(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return time.Time{}, fmt.Errorf("invalid PEM: %s", path)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}

	return cert.NotAfter, nil
}

func collect() {
	for _, c := range certs {
		expiry, err := readCertExpiry(c.Path)
		if err != nil {
			fmt.Printf("failed to read %s: %v\n", c.Path, err)
			continue
		}

		fmt.Printf("loaded cert %s expires %s\n",
			c.Name,
			expiry.Format(time.RFC3339),
		)

		certExpiry.WithLabelValues(
			c.Name,
			c.Group,
			c.Path,
		).Set(float64(expiry.Unix()))
	}
}

func metrics(w http.ResponseWriter, r *http.Request) {
	collect()
	promhttp.Handler().ServeHTTP(w, r)
}

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/config/certs.yaml"
	}

	loadedCerts, err := loadConfig(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	certs = loadedCerts

	http.HandleFunc("/metrics", metrics)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	fmt.Printf("loaded %d certificate targets\n", len(certs))
	fmt.Println("kubecert exporter running :9100")

	if err := http.ListenAndServe(":9100", nil); err != nil {
		panic(err)
	}
}