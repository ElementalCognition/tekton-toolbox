package clusterinterceptorupdater

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	testclient "k8s.io/client-go/kubernetes/fake"
)

const (
	svcName = "f22-rapror"
	tknNS   = "tekton-test-namespace"
)

func tlsVerify(cert, cacert []byte) error {
	certPool := x509.NewCertPool()
	pemCert, _ := pem.Decode(cert)
	certPool.AppendCertsFromPEM(cacert)
	leafCert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return err
	}
	vo := x509.VerifyOptions{
		DNSName:     fmt.Sprintf("%s.%s.svc", svcName, tknNS),
		Roots:       certPool,
		CurrentTime: time.Now().AddDate(9, 11, 30),
	}
	_, err = leafCert.Verify(vo)
	if err != nil {
		return err
	}
	return nil
}

func TestGetNamespace(t *testing.T) {
	err := os.Setenv("SVC_NAMESPACE", tknNS)
	assert.Nil(t, err)
	ns := GetNamespace()
	assert.Equal(t, tknNS, ns)
}

func TestGenerateCertificates(t *testing.T) {
	_, crt, cacert, err := generateCertificates(context.Background(), svcName, tknNS)
	assert.Nil(t, err)
	assert.Nil(t, tlsVerify(crt, cacert))
}

func TestGetCreateCertsSecret(t *testing.T) {
	client := testclient.NewSimpleClientset()
	logger := zaptest.NewLogger(t).Sugar()
	secret, err := getCreateCertsSecret(context.Background(), client.CoreV1(), logger, svcName, tknNS)
	assert.Nil(t, err)
	assert.Equal(t, svcName, secret.Name)
	assert.Equal(t, tknNS, secret.Namespace)
	assert.Nil(t, tlsVerify(secret.Data["server-cert.pem"], secret.Data["ca-cert.pem"]))
}
