package anchor_idl_parser

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// extractVector 解析一个动态长度的 vector，并返回逗号分隔的值字符串及字节数
func extractVector(data []byte, types []interface{}, offset int, argType interface{}) (string, int) {
	// 1. 读出长度（4 字节小端）
	length := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	n := 4

	// 2. 预分配 slice
	res := make([]string, 0, length)

	// 3. 循环提取并高效转换
	for i := 0; i < length; i++ {
		val, n_i := extractValue(data, types, offset+n, argType)
		n += n_i

		// 类型断言 & strconv 转换
		switch v := val.(type) {
		case string:
			res = append(res, v)
		case int:
			res = append(res, strconv.Itoa(v))
		case int64:
			res = append(res, strconv.FormatInt(v, 10))
		case uint64:
			res = append(res, strconv.FormatUint(v, 10))
		case float64:
			res = append(res, strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			res = append(res, strconv.FormatBool(v))
		default:
			// 最坏情况回落到 fmt.Sprint
			res = append(res, fmt.Sprint(v))
		}
	}

	// 4. 一次性拼接
	return strings.Join(res, ", "), n
}

// extractArray 解析定长 array，内部 args 类型由 IDL 给出
func extractArray(data []byte, types []interface{}, offset int, argType interface{}) (string, int) {
	// 1. 从 argType 中拿到 (elemType, length)
	meta := argType.([]interface{})
	rawLen := meta[1]
	length := 0
	switch v := rawLen.(type) {
	case int:
		length = v
	case float64:
		length = int(v)
	default:
		// 不常见类型：直接设置 0
		length = 0
	}

	n := 0
	res := make([]string, 0, length)

	// 2. 按长度循环
	for i := 0; i < length; i++ {
		val, n_i := extractValue(data, types, offset+n, meta[0])
		n += n_i

		switch v := val.(type) {
		case string:
			res = append(res, v)
		case int:
			res = append(res, strconv.Itoa(v))
		case int64:
			res = append(res, strconv.FormatInt(v, 10))
		case uint64:
			res = append(res, strconv.FormatUint(v, 10))
		case float64:
			res = append(res, strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			res = append(res, strconv.FormatBool(v))
		default:
			res = append(res, fmt.Sprint(v))
		}
	}

	return strings.Join(res, ", "), n
}
