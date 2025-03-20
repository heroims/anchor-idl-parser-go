# Anchor IDL Parser 
This is a parser for Solana programs compiled with Anchor using IDL.

## Install
```
go get github.com/heroims/anchor-idl-parser-go
```

## Usage
```
import (
	aip "github.com/heroims/anchor-idl-parser-go"
)

func main() {
	// Create Parser
    ammIdlParser, err := aip.NewParser("path/to/amm_idl.json")

	if err == nil {
		// Parse instruction (support cpi log)
	    insInfo, insErr := ammIdlParser.InstructionParse(instructionData)

        // Parse account
	    accountInfo, accErr := ammIdlParser.AccountsParse(accountData)

        // Parse log
	    eventInfo, eventErr := ammIdlParser.eventDataParse(logString)
	}

	
}
```
