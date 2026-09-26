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
)

// var dmHome = flag.String("DM_HOME", "", "Where DMDB installed")
var flagLock = sync.Mutex{}

func NewTLSFromTCP(conn net.Conn, sslCertPath, sslKeyPath, sslFilesPath, serverName string) (*tls.Conn, error) {
	if sslCertPath == "" || sslKeyPath == "" || sslFilesPath == "" {
		return nil, errors.New("SSL certificate, key, and CA directory are required")
	}
	cert, err := tls.LoadX509KeyPair(sslCertPath, sslKeyPath)
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
