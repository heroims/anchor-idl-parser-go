package anchor_idl_parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

func extractArgs(data []byte, args []interface{}, types []interface{}) map[string]interface{} {
	argsValues := make(map[string]interface{})
	offset := 0
	for _, arg := range args {
		argMap := arg.(map[string]interface{})
		argName := argMap["name"].(string)
		argType := argMap["type"]

		var n int
		argsValues[argName], n = extractValue(data, types, offset, argType)
		offset += n
	}
	return argsValues
}

func extractValue(data []byte, types []interface{}, offset int, argType interface{}) (interface{}, int) {
	pType, ok := argType.(string)
	if ok {
		return extractPrimitive(data, offset, pType)
	}

	npType, ok := argType.(map[string]interface{})
	if ok {
		return extractNonPrimitive(data, types, offset, npType)
	}

	return nil, 0
}

func extractNonPrimitive(data []byte, types []interface{}, offset int, argType map[string]interface{}) (interface{}, int) {
	vec, ok := argType["vec"]
	if ok {
		return extractVector(data, types, offset, vec)
	}
	arr, ok := argType["array"]
	if ok {
		return extractArray(data, types, offset, arr)
	}
	obj, ok := argType["defined"].(string)
	if ok {
		return extractObject(data, types, offset, obj)
	}
	opt, ok := argType["option"]
	if ok {
		return extractValue(data, types, offset, opt)
	}
	return nil, 0
}

func extractObject(data []byte, types []interface{}, offset int, typeName string) (string, int) {
	typeData, err := extractTypeData(types, typeName)
	if err != nil {
		return "", 0
	}
	switch typeData["kind"] {
	case "struct":
		return extractStruct(data, types, offset, typeData)
	case "enum":
		return extractEnum(data, types, offset, typeData)
	default:
		panic(fmt.Sprintf("that kind is not supported, kind: %s", typeData["kind"]))
	}

}

func extractStruct(data []byte, types []interface{}, offset int, typeData map[string]interface{}) (string, int) {
	fields := typeData["fields"].([]interface{})

	res := make(map[string]interface{})
	var n int = 0

	var n_i int
	for _, field := range fields {
		field, ok := field.(map[string]interface{})
		if !ok {
			log.Println("cannot cast field to map[string]interface{}, in extractObject")
		}
		res[field["name"].(string)], n_i = extractValue(data, types, offset+n, field["type"])
		n += n_i
	}

	// IDK maybe I should check it :)
	json, _ := json.Marshal(res)

	return string(json), n
}

func extractEnum(data []byte, types []interface{}, offset int, typeData map[string]interface{}) (string, int) {
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
		option, n_i := handleUnnamedEnumArgs(data, types, offset+n, fields)
		n += n_i

		res[memberName] = option
		// IDK maybe I should check err value here :)
		json, _ := json.Marshal(res)

		return string(json), n
	}

	f1, ok := fields[0].(map[string]interface{})
	_, hasName := f1["name"]
	if ok && !hasName {
		option, n_i := handleUnnamedEnumArgs(data, types, offset+n, fields)
		n += n_i

		res[memberName] = option
		// IDK maybe I should check err value here :)
		json, _ := json.Marshal(res)

		return string(json), n
	}

	option, n_i := handleNamedEnumArgs(data, types, offset+n, fields)
	n += n_i

	res[memberName] = option

	// IDK maybe I should check err value here :)
	json, _ := json.Marshal(res)

	return string(json), n
}

func handleNamedEnumArgs(data []byte, types []interface{}, offset int, fields []interface{}) (interface{}, int) {
	n := 0
	var n_i int
	option := make(map[string]interface{})
	for _, field := range fields {
		obj, ok := field.(map[string]interface{})
		if ok {
			option[obj["name"].(string)], n_i = extractValue(data, types, offset+n, obj["type"])
			n += n_i
			continue
		}
	}
	return option, n
}
func handleUnnamedEnumArgs(data []byte, types []interface{}, offset int, fields []interface{}) (interface{}, int) {
	n := 0
	var n_i int
	option := make([]interface{}, len(fields))
	for i, field := range fields {
		option[i], n_i = extractValue(data, types, offset+n, field)
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
