package utils

import (
	"crypto/tls"
	"log"
	"os"

	"go.temporal.io/sdk/client"
)

func GetTemporalClient() (client.Client, error) {
	TemporalHostPort := os.Getenv("TEMPORAL_HOST_PORT")
	TemporalNamespace := os.Getenv("TEMPORAL_NAMESPACE")
	TemporalCert := os.Getenv("TEMPORAL_CERT")
	TemporalCertKey := os.Getenv("TEMPORAL_CERT_KEY")

	cert, err := tls.X509KeyPair([]byte(TemporalCert), []byte(TemporalCertKey))
	if err != nil {
		log.Fatalln("Unable to load cert and key pair.", err)
	}

	return client.Dial(client.Options{
		HostPort:  TemporalHostPort,
		Namespace: TemporalNamespace,
		ConnectionOptions: client.ConnectionOptions{
			TLS: &tls.Config{Certificates: []tls.Certificate{cert}},
		},
	})
}
