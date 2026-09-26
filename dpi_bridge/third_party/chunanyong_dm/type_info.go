package dm

// ArrayTypeInfo exposes the parts of the server's type descriptor needed by
// clients that bind and fetch user-defined arrays.
type ArrayTypeInfo struct {
	Kind             int
	ElementKind      int
	ElementPrecision int
	ElementScale     int
	MaxElements      int
}

func DescribeArrayType(conn *DmConnection, typeName string) (ArrayTypeInfo, error) {
	desc, err := newArrayDescriptor(typeName, conn)
	if err != nil {
		return ArrayTypeInfo{}, err
	}
	root := desc.getMDesc()
	item := desc.getItemDesc()
	if root == nil || item == nil {
		return ArrayTypeInfo{}, ECGO_INVALID_PARAMETER_VALUE.throw()
	}
	maxElements := root.getMaxCnt()
	kind := root.getDType()
	if kind == CLASS && root.getObjId() == 4 &&
		(root.getCltnType() == CLTN_TYPE_VARRAY || root.getCltnType() == CLTN_TYPE_NST_TABLE) {
		kind = ARRAY
	}
	if kind == SARRAY {
		maxElements = root.getStaticArrayLength()
	}
	return ArrayTypeInfo{
		Kind:             kind,
		ElementKind:      item.getDType(),
		ElementPrecision: item.getPrec(),
		ElementScale:     item.getScale(),
		MaxElements:      maxElements,
	}, nil
}
