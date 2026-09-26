package dm

import "testing"

func TestComplexArrayRejectsInvalidElementWithoutPanic(t *testing.T) {
	for _, kind := range []int{CLASS, ARRAY} {
		root := newTypeDescriptor(nil)
		root.m_arrObj = newTypeDescriptor(nil)
		root.m_arrObj.column.colType = int32(kind)
		if kind == ARRAY {
			root.m_arrObj.m_arrObj = newTypeDescriptor(nil)
			root.m_arrObj.m_arrObj.column.colType = INT
		}
		if _, err := TypeDataSV.toArray([]interface{}{"invalid"}, root); err == nil {
			t.Errorf("kind %d accepted an invalid nested element", kind)
		}
	}
}
