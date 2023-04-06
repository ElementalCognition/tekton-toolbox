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

var (
	svcName = "f22-raptor"
	tknNS   = "tekton-test-namespace"
)

func tlsVerify(t *testing.T, cert, cacert []byte) error {
	certPool := x509.NewCertPool()
	pemCert, _ := pem.Decode(cert)
	certPool.AppendCertsFromPEM(cacert)
	leafCert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		t.Error(err)
	}
	vo := x509.VerifyOptions{
		DNSName:     fmt.Sprintf("%s.%s.svc", svcName, tknNS),
		Roots:       certPool,
		CurrentTime: time.Now().AddDate(9, 11, 30),
	}
	_, err = leafCert.Verify(vo)
	if err != nil {
		t.Error(err)
	}
	return err
}

func TestGetNamespace(t *testing.T) {
	err := os.Setenv("SVC_NAMESPACE", tknNS)
	assert.Nil(t, err)
	assert.Equal(t, tknNS, GetNamespace())
	os.Unsetenv("SVC_NAMESPACE")
	assert.Equal(t, "tekton-pipelines", GetNamespace())
}

func TestGenerateCertificates(t *testing.T) {
	key, crt, cacert, err := generateCertificates(context.Background(), svcName, tknNS)
	assert.Nil(t, err)
	assert.IsType(t, []byte{}, key)
	assert.Nil(t, tlsVerify(t, crt, cacert))

	_, _, _, err = generateCertificates(context.Background(), "", tknNS)
	assert.Error(t, err)
	_, _, _, err = generateCertificates(context.Background(), "", "")
	assert.Error(t, err)
	_, _, _, err = generateCertificates(context.Background(), "/*-/*12", "9ф?:^")
	assert.Error(t, err)
}

func TestGetCreateCertsSecret(t *testing.T) {
	client := testclient.NewSimpleClientset()
	logger := zaptest.NewLogger(t).Sugar()
	secret, err := getCreateCertsSecret(context.Background(), client.CoreV1(), logger, svcName, tknNS)
	assert.Nil(t, err)
	assert.Equal(t, svcName, secret.Name)
	assert.Equal(t, tknNS, secret.Namespace)
	assert.Nil(t, tlsVerify(t, secret.Data["server-cert.pem"], secret.Data["ca-cert.pem"]))
}
