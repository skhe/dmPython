package dm

// CompressionMode reports the negotiated wire-compression mode to the DPI bridge.
func (dc *DmConnection) CompressionMode() int {
	if dc.dmConnector == nil {
		return 0
	}
	return dc.dmConnector.compress
}
