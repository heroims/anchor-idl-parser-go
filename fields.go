package anchor_idl_parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

const maxRecursiveDepth = 62

func extractArgs(data []byte, args []interface{}, types []interface{}) map[string]interface{} {
	return extractArgsWithDepth(data, args, types, 0)
}

func extractArgsWithDepth(data []byte, args []interface{}, types []interface{}, depth int) map[string]interface{} {
	argsValues := make(map[string]interface{})
	offset := 0
	for _, arg := range args {
		argMap := arg.(map[string]interface{})
		argName := argMap["name"].(string)
		argType := argMap["type"]

		var n int
		argsValues[argName], n = extractValueWithDepth(data, types, offset, argType, depth)
		offset += n
	}
	return argsValues
}

func extractValue(data []byte, types []interface{}, offset int, argType interface{}) (interface{}, int) {
	return extractValueWithDepth(data, types, offset, argType, 0)
}

func extractValueWithDepth(data []byte, types []interface{}, offset int, argType interface{}, depth int) (interface{}, int) {
	if depth > maxRecursiveDepth {
		return nil, 0
	}
	pType, ok := argType.(string)
	if ok {
		return extractPrimitive(data, offset, pType)
	}

	npType, ok := argType.(map[string]interface{})
	if ok {
		return extractNonPrimitiveWithDepth(data, types, offset, npType, depth+1)
	}

	return nil, 0
}

func extractNonPrimitiveWithDepth(data []byte, types []interface{}, offset int, argType map[string]interface{}, depth int) (interface{}, int) {
	if depth > maxRecursiveDepth {
		return nil, 0
	}
	vec, ok := argType["vec"]
	if ok {
		return extractVector(data, types, offset, vec)
	}
	arr, ok := argType["array"]
	if ok {
		return extractArray(data, types, offset, arr)
	}
	obj, ok := argType["defined"]
	if ok {
		value, ok := obj.(string)
		if ok {
			return extractObjectWithDepth(data, types, offset, value, depth+1)
		} else {
			value, ok := obj.(map[string]interface{})
			if ok {
				name, ok := value["name"]
				if ok {
					if value, ok := name.(string); ok {
						return extractObjectWithDepth(data, types, offset, value, depth+1)
					}
				}
			}
		}
	}
	opt, ok := argType["option"]
	if ok {
		return extractValueWithDepth(data, types, offset, opt, depth+1)
	}
	return nil, 0
}

func extractObjectWithDepth(data []byte, types []interface{}, offset int, typeName string, depth int) (string, int) {
	if depth > maxRecursiveDepth {
		return "", 0
	}
	typeData, err := extractTypeData(types, typeName)
	if err != nil {
		return "", 0
	}
	switch typeData["kind"] {
	case "struct":
		return extractStructWithDepth(data, types, offset, typeData, depth+1)
	case "enum":
		return extractEnumWithDepth(data, types, offset, typeData, depth+1)
	default:
		panic(fmt.Sprintf("that kind is not supported, kind: %s", typeData["kind"]))
	}
}

func extractStructWithDepth(data []byte, types []interface{}, offset int, typeData map[string]interface{}, depth int) (string, int) {
	if depth > maxRecursiveDepth {
		return "", 0
	}
	fields := typeData["fields"].([]interface{})

	res := make(map[string]interface{})
	var n int = 0

	var n_i int
	for _, field := range fields {
		field, ok := field.(map[string]interface{})
		if !ok {
			log.Println("cannot cast field to map[string]interface{}, in extractObject")
		}
		res[field["name"].(string)], n_i = extractValueWithDepth(data, types, offset+n, field["type"], depth+1)
		n += n_i
	}

	json, _ := json.Marshal(res)
	return string(json), n
}

func extractEnumWithDepth(data []byte, types []interface{}, offset int, typeData map[string]interface{}, depth int) (string, int) {
	if depth > maxRecursiveDepth {
		return "", 0
	}
	variants := typeData["variants"].([]interface{})
	variantId := data[offset]
	variant := variants[variantId].(map[string]interface{})

	memberName := variant["name"].(string)

	res := make(map[string]interface{})

	fields, ok := variant["fields"].([]interface{})
	if !ok {
		res[memberName] = make(map[string]interface{})
		json, _ := json.Marshal(res)
		return string(json), 1
	}

	var n int = 1

	_, ok = fields[0].(string)
	if ok {
		option, n_i := handleUnnamedEnumArgsWithDepth(data, types, offset+n, fields, depth+1)
		n += n_i

		res[memberName] = option
		json, _ := json.Marshal(res)
		return string(json), n
	}

	f1, ok := fields[0].(map[string]interface{})
	_, hasName := f1["name"]
	if ok && !hasName {
		option, n_i := handleUnnamedEnumArgsWithDepth(data, types, offset+n, fields, depth+1)
		n += n_i

		res[memberName] = option
		json, _ := json.Marshal(res)
		return string(json), n
	}

	option, n_i := handleNamedEnumArgsWithDepth(data, types, offset+n, fields, depth+1)
	n += n_i

	res[memberName] = option
	json, _ := json.Marshal(res)
	return string(json), n
}

func handleNamedEnumArgsWithDepth(data []byte, types []interface{}, offset int, fields []interface{}, depth int) (interface{}, int) {
	if depth > maxRecursiveDepth {
		return nil, 0
	}
	n := 0
	var n_i int
	option := make(map[string]interface{})
	for _, field := range fields {
		obj, ok := field.(map[string]interface{})
		if ok {
			option[obj["name"].(string)], n_i = extractValueWithDepth(data, types, offset+n, obj["type"], depth+1)
			n += n_i
			continue
		}
	}
	return option, n
}

func handleUnnamedEnumArgsWithDepth(data []byte, types []interface{}, offset int, fields []interface{}, depth int) (interface{}, int) {
	if depth > maxRecursiveDepth {
		return nil, 0
	}
	n := 0
	var n_i int
	option := make([]interface{}, len(fields))
	for i, field := range fields {
		option[i], n_i = extractValueWithDepth(data, types, offset+n, field, depth+1)
		n += n_i
	}
	return option, n
}

func extractTypeData(types []interface{}, typeName string) (map[string]interface{}, error) {
	for _, t := range types {
		t, ok := t.(map[string]interface{})
		if !ok {
			return nil, errors.New("cannot cast type to map[string]interface{}")
		}
		if strings.EqualFold(t["name"].(string), typeName) {
			return t["type"].(map[string]interface{}), nil
		}
	}
	return nil, fmt.Errorf("couldn't find type: %s, in: %+v", typeName, types)
}
