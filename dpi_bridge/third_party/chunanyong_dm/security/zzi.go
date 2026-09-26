/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/youmark/pkcs8"
)

// var dmHome = flag.String("DM_HOME", "", "Where DMDB installed")
var flagLock = sync.Mutex{}

func NewTLSFromTCP(conn net.Conn, sslCertPath, sslKeyPath, sslFilesPath, serverName, sslKeyPassword string) (*tls.Conn, error) {
	if sslCertPath == "" || sslKeyPath == "" || sslFilesPath == "" {
		return nil, errors.New("SSL certificate, key, and CA directory are required")
	}
	cert, err := loadClientKeyPair(sslCertPath, sslKeyPath, sslKeyPassword)
	if err != nil {
		return nil, err
	}
	caPEM, err := os.ReadFile(filepath.Join(sslFilesPath, "ca-cert.pem"))
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("SSL CA file has no valid certificates")
	}
	conf := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ServerName:   serverName,
		Certificates: []tls.Certificate{cert},
		// DM's bundled server certificate has no SAN and uses SHA1. Require an
		// exact certificate pin for that legacy case; modern certs use CA and SAN.
		InsecureSkipVerify: true,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return errors.New("SSL server did not provide a certificate")
			}
			leaf := state.PeerCertificates[0]
			if len(leaf.DNSNames) == 0 && len(leaf.IPAddresses) == 0 {
				pinnedPEM, err := os.ReadFile(filepath.Join(sslFilesPath, "server-cert.pem"))
				if err != nil {
					return fmt.Errorf("legacy SSL server certificate requires server-cert.pem pin: %w", err)
				}
				block, _ := pem.Decode(pinnedPEM)
				if block == nil || !bytes.Equal(leaf.Raw, block.Bytes) {
					return errors.New("SSL server certificate does not match server-cert.pem pin")
				}
				now := time.Now()
				if now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
					return errors.New("SSL server certificate pin is expired or not yet valid")
				}
				if leaf.IsCA || (leaf.Subject.CommonName != "server" && leaf.Subject.CommonName != serverName) {
					return fmt.Errorf("SSL server certificate has unexpected CN %q", leaf.Subject.CommonName)
				}
				return nil
			}
			intermediates := x509.NewCertPool()
			for _, intermediate := range state.PeerCertificates[1:] {
				intermediates.AddCert(intermediate)
			}
			if _, err := leaf.Verify(x509.VerifyOptions{
				Roots: roots, Intermediates: intermediates,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}); err != nil {
				return err
			}
			return leaf.VerifyHostname(serverName)
		},
	}
	tlsConn := tls.Client(conn, conf)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}
	return tlsConn, nil
}

func loadClientKeyPair(certPath, keyPath, password string) (tls.Certificate, error) {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return tls.Certificate{}, errors.New("SSL private key is not PEM encoded")
	}
	if block.Type == "ENCRYPTED PRIVATE KEY" {
		if password == "" {
			return tls.Certificate{}, errors.New("encrypted SSL private key requires ssl_pwd")
		}
		privateKey, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(password))
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("failed to decrypt PKCS#8 SSL private key: %w", err)
		}
		der, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return tls.Certificate{}, err
		}
		defer func() {
			for i := range der {
				der[i] = 0
			}
		}()
		certPEM, err := os.ReadFile(certPath)
		if err != nil {
			return tls.Certificate{}, err
		}
		plainKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
		defer func() {
			for i := range plainKeyPEM {
				plainKeyPEM[i] = 0
			}
		}()
		return tls.X509KeyPair(certPEM, plainKeyPEM)
	}
	if !x509.IsEncryptedPEMBlock(block) {
		return tls.LoadX509KeyPair(certPath, keyPath)
	}
	if password == "" {
		return tls.Certificate{}, errors.New("encrypted SSL private key requires ssl_pwd")
	}
	der, err := x509.DecryptPEMBlock(block, []byte(password))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to decrypt SSL private key: %w", err)
	}
	defer func() {
		for i := range der {
			der[i] = 0
		}
	}()
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(certPEM, pem.EncodeToMemory(&pem.Block{Type: block.Type, Bytes: der}))
}
