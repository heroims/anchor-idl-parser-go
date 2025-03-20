package anchor_idl_parser

import (
	"encoding/binary"
	"fmt"
	"strings"
)

func extractVector(data []byte, types []interface{}, offset int, argType interface{}) (string, int) {
	res := make([]string, 0)
	len := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	var n int = 4
	for range len {
		val, n_i := extractValue(data, types, int(offset)+n, argType)
		res = append(res, fmt.Sprint(val))
		n += n_i
	}
	return strings.Join(res, ", "), n
}

func extractArray(data []byte, types []interface{}, offset int, argType interface{}) (string, int) {

	res := make([]string, 0)

	typeData := argType.([]interface{})
	len, ok := typeData[1].(int)
	if !ok {
		// idk why but it says that it should be float64 when parsing array inside an object
		len = int(typeData[1].(float64))
	}

	var n int = 0
	for range len {
		val, n_i := extractValue(data, types, offset+n, typeData[0])
		res = append(res, fmt.Sprint(val))
		n += n_i
	}

	return strings.Join(res, ", "), n
}
