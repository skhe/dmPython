package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testCertificate(t *testing.T, template, parent *x509.Certificate, signer *ecdsa.PrivateKey) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if signer == nil {
		signer = key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, signer)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), key
}

func testCA(t *testing.T) ([]byte, *x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	now := time.Now()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), cert, key
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func testTLSFiles(t *testing.T) (string, tls.Certificate) {
	t.Helper()
	dir := t.TempDir()
	caPEM, ca, caKey := testCA(t)
	writeTestFile(t, filepath.Join(dir, "ca-cert.pem"), caPEM)
	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(2), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("::1")},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafPEM, leafKey := testCertificate(t, leaf, ca, caKey)
	keyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	writeTestFile(t, filepath.Join(dir, "client-cert.pem"), leafPEM)
	writeTestFile(t, filepath.Join(dir, "client-key.pem"), keyPEM)
	serverCert, err := tls.X509KeyPair(leafPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return dir, serverCert
}

func testTLSHandshake(t *testing.T, dir string, serverCert tls.Certificate, serverName string) error {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	serverDone := make(chan error, 1)
	go func() {
		serverSocket, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		server := tls.Server(serverSocket, &tls.Config{Certificates: []tls.Certificate{serverCert}, MinVersion: tls.VersionTLS12})
		serverDone <- server.Handshake()
		server.Close()
	}()
	clientSocket, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewTLSFromTCP(clientSocket, filepath.Join(dir, "client-cert.pem"), filepath.Join(dir, "client-key.pem"), dir, serverName)
	if client != nil {
		client.Close()
	} else {
		clientSocket.Close()
	}
	<-serverDone
	return err
}

func TestModernServerCertificateVerification(t *testing.T) {
	dir, serverCert := testTLSFiles(t)
	if err := testTLSHandshake(t, dir, serverCert, "localhost"); err != nil {
		t.Fatalf("trusted DNS name: %v", err)
	}
	if err := testTLSHandshake(t, dir, serverCert, "[::1]"); err != nil {
		t.Fatalf("trusted bracketed IPv6: %v", err)
	}
	if err := testTLSHandshake(t, dir, serverCert, "wrong.example"); err == nil || !strings.Contains(err.Error(), "not wrong.example") {
		t.Fatalf("wrong hostname should fail verification, got %v", err)
	}
	untrustedCA, _, _ := testCA(t)
	writeTestFile(t, filepath.Join(dir, "ca-cert.pem"), untrustedCA)
	if err := testTLSHandshake(t, dir, serverCert, "localhost"); err == nil || !strings.Contains(err.Error(), "unknown authority") {
		t.Fatalf("untrusted CA should fail verification, got %v", err)
	}
}
