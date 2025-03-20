package anchor_idl_parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/heroims/anchor-idl-parser-go/utils"
)

type Parser struct {
	idlPath string
	idlJson string
	idlMap  map[string]interface{}
}

func NewParser(idlPath string) (*Parser, error) {
	idlData, err := os.ReadFile(idlPath)
	if err != nil {
		return nil, err
	}
	idlJson := string(idlData)
	var idlMap map[string]interface{}
	err = json.Unmarshal([]byte(idlJson), &idlMap)
	if err != nil {
		return nil, err
	}
	return &Parser{
		idlPath: idlPath,
		idlJson: idlJson,
		idlMap:  idlMap,
	}, nil
}

func (p *Parser) InstructionParse(data []byte) (map[string]interface{}, error) {
	if len(data) >= 8 {
		hexStr := "1d9acb512ea545e4"
		cpiDiscriminatorBytes, err := hex.DecodeString(hexStr)
		cpiDiscriminatorBytes = utils.ReverseBytes(cpiDiscriminatorBytes)
		if err != nil {
			return nil, errors.New("DecodeString failed")
		}
		if bytes.Equal(data[:8], cpiDiscriminatorBytes) {
			return p.cpiEventParse(data[8:])
		}
	}

	instructions, ok := p.idlMap["instructions"].([]interface{})
	if !ok {
		return nil, errors.New("instructions not found in IDL")
	}

	types, ok := p.idlMap["types"].([]interface{})
	if !ok {
		return nil, errors.New("types not found in IDL")
	}

	for _, instruction := range instructions {
		instructionMap, ok := instruction.(map[string]interface{})
		if !ok {
			continue
		}
		if discriminator, ok := instructionMap["discriminator"].([]interface{}); ok {
			if len(discriminator) == 4 {
				discriminatorBytes := make([]byte, 4)
				for i, val := range discriminator {
					discriminatorBytes[i] = byte(val.(float64))
				}

				if bytes.Equal(data[:4], discriminatorBytes) {
					argsValues := make(map[string]interface{})
					argsValues["discriminator"] = instructionMap["discriminator"]
					argsValues["args"] = extractArgs(data[4:], instructionMap["args"].([]interface{}), types)
					return argsValues, nil
				}
			}
		} else {
			instructionName, ok := instructionMap["name"].(string)

			if !ok {
				continue
			}
			instructionName = utils.ToSnakeCase(instructionName)
			hash := sha256.Sum256([]byte(fmt.Sprintf("global:%s", instructionName)))

			if bytes.Equal(data[:8], hash[:8]) {
				argsValues := make(map[string]interface{})
				argsValues["name"] = instructionMap["name"]
				argsValues["args"] = extractArgs(data[8:], instructionMap["args"].([]interface{}), types)
				return argsValues, nil
			}
		}
	}
	return nil, errors.New("can't find instruction")
}

func (p *Parser) EventParse(log string) (map[string]interface{}, error) {
	PROGRAM_LOG := "Program log: "
	PROGRAM_DATA := "Program data: "
	PROGRAM_LOG_START_INDEX := len(PROGRAM_LOG)
	PROGRAM_DATA_START_INDEX := len(PROGRAM_DATA)
	var logStr string
	if strings.HasPrefix(log, PROGRAM_LOG) {
		logStr = log[PROGRAM_LOG_START_INDEX:]
	} else if strings.HasPrefix(log, PROGRAM_DATA) {
		logStr = log[PROGRAM_DATA_START_INDEX:]
	} else {
		return nil, errors.New("log does not start with a valid prefix")
	}

	decoded, err := base64.StdEncoding.DecodeString(logStr)
	if err != nil {
		return nil, errors.New("failed to decode base64 log string")
	}

	return p.eventDataParse(decoded)
}

func (p *Parser) eventDataParse(data []byte) (map[string]interface{}, error) {
	events, ok := p.idlMap["events"].([]interface{})
	if !ok {
		return nil, errors.New("events not found in IDL")
	}

	types, ok := p.idlMap["types"].([]interface{})
	if !ok {
		return nil, errors.New("types not found in IDL")
	}

	for _, event := range events {
		eventMap, ok := event.(map[string]interface{})
		if !ok {
			continue
		}
		if discriminator, ok := eventMap["discriminator"].([]interface{}); ok {
			if len(discriminator) == 4 {
				discriminatorBytes := make([]byte, 4)
				for i, val := range discriminator {
					discriminatorBytes[i] = byte(val.(float64))
				}

				if bytes.Equal(data[:4], discriminatorBytes) {
					argsValues := make(map[string]interface{})
					argsValues["discriminator"] = eventMap["discriminator"]
					argsValues["args"] = extractArgs(data[4:], eventMap["args"].([]interface{}), types)
					return argsValues, nil
				}
			}
		} else {
			eventName, ok := eventMap["name"].(string)

			if !ok {
				continue
			}
			hash := sha256.Sum256([]byte(fmt.Sprintf("event:%s", eventName)))

			if bytes.Equal(data[:8], hash[:8]) {
				argsValues := make(map[string]interface{})
				argsValues["name"] = eventName
				argsValues["fields"] = extractArgs(data[8:], eventMap["fields"].([]interface{}), types)
				return argsValues, nil
			}
		}
	}
	return nil, errors.New("can't find event")
}

func (p *Parser) cpiEventParse(data []byte) (map[string]interface{}, error) {
	return p.eventDataParse(data)
}
