package dm

import "errors"

// CompressionMode reports the negotiated wire-compression mode to the DPI bridge.
func (dc *DmConnection) CompressionMode() int {
	if dc.dmConnector == nil {
		return 0
	}
	return dc.dmConnector.compress
}

// SSLMode reports the server-negotiated TLS mode (1 encrypts the session).
func (dc *DmConnection) SSLMode() int {
	return dc.sslEncrypt
}

// SetAutoCommit changes the mode carried on subsequent wire requests.
// Enabling autocommit commits any pending manual transaction first.
func (dc *DmConnection) SetAutoCommit(enabled bool) error {
	if dc.dmConnector == nil {
		return errors.New("connection is not initialized")
	}
	if err := dc.checkClosed(); err != nil {
		return err
	}
	if enabled && !dc.autoCommit {
		if err := dc.commit(); err != nil {
			return err
		}
	}
	dc.dmConnector.autoCommit = enabled
	dc.autoCommit = enabled
	return nil
}
