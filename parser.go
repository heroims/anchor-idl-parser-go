package anchor_idl_parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/heroims/anchor-idl-parser-go/utils"
)

type Parser struct {
	idlPath string
	idlJson string
	idlMap  map[string]interface{}
}

func (p *Parser) GetIdlMap() map[string]interface{} {
	return p.idlMap
}

func (p *Parser) GetIdlJson() string {
	return p.idlJson
}

func (p *Parser) GetIdlPath() string {
	return p.idlPath
}

func NewParserWithPath(idlPath string) (*Parser, error) {
	idlData, err := os.ReadFile(idlPath)
	if err != nil {
		return nil, err
	}
	idlJson := string(idlData)
	var idlMap map[string]interface{}
	err = sonic.Unmarshal([]byte(idlJson), &idlMap)
	if err != nil {
		return nil, err
	}
	return &Parser{
		idlPath: idlPath,
		idlJson: idlJson,
		idlMap:  idlMap,
	}, nil
}

func NewParserWithJson(idlJson string) (*Parser, error) {
	var idlMap map[string]interface{}
	err := sonic.Unmarshal([]byte(idlJson), &idlMap)
	if err != nil {
		return nil, err
	}
	return &Parser{
		idlPath: "",
		idlJson: idlJson,
		idlMap:  idlMap,
	}, nil
}

func NewParserWithJsonMap(idlMap map[string]interface{}) (*Parser, error) {
	jsonBytes, err := sonic.Marshal(idlMap)
	if err != nil {
		panic(err)
	}
	return &Parser{
		idlPath: "",
		idlJson: string(jsonBytes),
		idlMap:  idlMap,
	}, nil
}

func (p *Parser) InstructionParse(data []byte) (map[string]interface{}, error) {
	if len(data) < 8 {
		return nil, errors.New("invalid data length")
	}

	hexStr := "1d9acb512ea545e4"
	cpiDiscriminatorBytes, err := hex.DecodeString(hexStr)
	cpiDiscriminatorBytes = utils.ReverseBytes(cpiDiscriminatorBytes)
	if err != nil {
		return nil, errors.New("DecodeString failed")
	}
	if bytes.Equal(data[:8], cpiDiscriminatorBytes) {
		return p.cpiEventParse(data[8:])
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
			discriminatorBytesLen := len(discriminator)

			discriminatorBytes := make([]byte, discriminatorBytesLen)
			for i, val := range discriminator {
				if valValue, ok := val.(float64); ok {
					discriminatorBytes[i] = byte(valValue)
				}
			}

			if bytes.Equal(data[:discriminatorBytesLen], discriminatorBytes) {
				argsValues := make(map[string]interface{})
				argsValues["name"] = instructionMap["name"]
				argsValues["discriminator"] = instructionMap["discriminator"]
				if argsValue, ok := instructionMap["args"].([]interface{}); ok {
					argsValues["data"] = extractArgs(data[discriminatorBytesLen:], argsValue, types)
				}
				argsValues["type"] = "instruction"
				return argsValues, nil
			}
		} else {
			instructionName, ok := instructionMap["name"].(string)

			if !ok {
				continue
			}
			instructionName = utils.ToSnakeCase(instructionName)
			hash := sha256.Sum256([]byte("global:" + instructionName))

			if bytes.Equal(data[:8], hash[:8]) {
				argsValues := make(map[string]interface{})
				argsValues["name"] = instructionMap["name"]
				argsValues["data"] = extractArgs(data[8:], instructionMap["args"].([]interface{}), types)
				argsValues["type"] = "instruction"
				return argsValues, nil
			}
		}
	}
	return nil, errors.New("can't find instruction")
}

func (p *Parser) AccountsParse(data []byte) (map[string]interface{}, error) {
	accounts, ok := p.idlMap["accounts"].([]interface{})
	if !ok {
		return nil, errors.New("accounts not found in IDL")
	}

	types, ok := p.idlMap["types"].([]interface{})
	if !ok {
		return nil, errors.New("types not found in IDL")
	}

	for _, account := range accounts {
		accountMap, ok := account.(map[string]interface{})
		if !ok {
			continue
		}
		if discriminator, ok := accountMap["discriminator"].([]interface{}); ok {
			discriminatorBytesLen := len(discriminator)

			discriminatorBytes := make([]byte, discriminatorBytesLen)
			for i, val := range discriminator {
				if valValue, ok := val.(float64); ok {
					discriminatorBytes[i] = byte(valValue)
				}
			}
			if bytes.Equal(data[:discriminatorBytesLen], discriminatorBytes) {
				argsValues := make(map[string]interface{})
				argsValues["discriminator"] = discriminator
				var accountArgs []interface{}
				for _, typeVal := range types {
					typeMap, ok := typeVal.(map[string]interface{})
					if !ok {
						continue
					}
					if typeMap["name"] == accountMap["name"] {
						if typeDetails, ok := typeMap["type"].(map[string]interface{}); ok {
							if tmpAccountArgs, ok := typeDetails["fields"].([]interface{}); ok {
								accountArgs = tmpAccountArgs
							}
						}
						break
					}
				}
				argsValues["data"] = extractArgs(data[discriminatorBytesLen:], accountArgs, types)
				argsValues["type"] = "account"
				return argsValues, nil
			}
		} else {
			accountName, ok := accountMap["name"].(string)

			if !ok {
				continue
			}
			hash := sha256.Sum256([]byte("account:" + accountName))

			if bytes.Equal(data[:8], hash[:8]) {
				argsValues := make(map[string]interface{})
				argsValues["name"] = accountMap["name"]
				var accountArgs []interface{}
				if accountType, ok := accountMap["type"].(map[string]interface{}); ok {
					if tmpAccountArgs, ok := accountType["fields"].([]interface{}); ok {
						accountArgs = tmpAccountArgs
					}
				}
				argsValues["data"] = extractArgs(data[8:], accountArgs, types)
				argsValues["type"] = "account"
				return argsValues, nil
			}
		}
	}
	return nil, errors.New("can't find accounts")
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
			discriminatorBytesLen := len(discriminator)

			discriminatorBytes := make([]byte, discriminatorBytesLen)
			for i, val := range discriminator {
				if valValue, ok := val.(float64); ok {
					discriminatorBytes[i] = byte(valValue)
				}
			}

			if bytes.Equal(data[:discriminatorBytesLen], discriminatorBytes) {
				argsValues := make(map[string]interface{})
				argsValues["name"] = eventMap["name"]
				argsValues["discriminator"] = eventMap["discriminator"]
				var eventArgs []interface{}
				for _, typeVal := range types {
					typeMap, ok := typeVal.(map[string]interface{})
					if !ok {
						continue
					}
					if typeMap["name"] == eventMap["name"] {
						if typeDetails, ok := typeMap["type"].(map[string]interface{}); ok {
							if tmpEventArgs, ok := typeDetails["fields"].([]interface{}); ok {
								eventArgs = tmpEventArgs
							}
						}
						break
					}
				}
				argsValues["data"] = extractArgs(data[discriminatorBytesLen:], eventArgs, types)
				argsValues["type"] = "event"
				return argsValues, nil
			}
		} else {
			eventName, ok := eventMap["name"].(string)

			if !ok {
				continue
			}
			hash := sha256.Sum256([]byte("event:" + eventName))

			if bytes.Equal(data[:8], hash[:8]) {
				argsValues := make(map[string]interface{})
				argsValues["name"] = eventName
				if filedValue, ok := eventMap["fields"].([]interface{}); ok {
					argsValues["data"] = extractArgs(data[8:], filedValue, types)
				}
				argsValues["type"] = "event"
				return argsValues, nil
			}
		}
	}
	return nil, errors.New("can't find event")
}

func (p *Parser) cpiEventParse(data []byte) (map[string]interface{}, error) {
	return p.eventDataParse(data)
}
