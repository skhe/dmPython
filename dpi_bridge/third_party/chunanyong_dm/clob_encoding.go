package dm

// gb18030SafePrefix returns the longest whole-character prefix within limit.
// The CLOB PUT_DATA protocol must not split a two- or four-byte character.
func gb18030SafePrefix(buffer *Dm_build_0, limit int) int {
	position := 0
	for position < limit {
		width := 1
		first := buffer.dm_build_32(position)
		if first >= 0x81 && first <= 0xFE {
			if position+1 >= limit {
				break
			}
			second := buffer.dm_build_32(position + 1)
			if second >= 0x30 && second <= 0x39 {
				width = 4
			} else {
				width = 2
			}
		}
		if position+width > limit {
			break
		}
		position += width
	}
	return position
}
