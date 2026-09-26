package dm

import "testing"

func TestGB18030SafePrefix(t *testing.T) {
	// ASCII, two-byte Chinese, four-byte supplementary character, ASCII.
	encoded := []byte{'A', 0xD6, 0xD0, 0x95, 0x30, 0x8B, 0x34, 'B'}
	buffer := Dm_build_4()
	buffer.Dm_build_26(encoded, 0, len(encoded))
	want := []int{0, 1, 1, 3, 3, 3, 3, 7, 8}
	for limit, expected := range want {
		if got := gb18030SafePrefix(buffer, limit); got != expected {
			t.Errorf("limit %d: got %d, want %d", limit, got, expected)
		}
	}
}
