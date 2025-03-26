package anchor_idl_parser

import (
	"encoding/binary"
	"math"
	"math/big"

	"github.com/btcsuite/btcutil/base58"
)

func extractPrimitive(data []byte, offset int, argType string) (interface{}, int) {
	switch argType {
	case "u128":
		if len(data[offset:]) < 16 {
			return nil, 16
		} else {
			b := data[offset : offset+16]
			bigInt := new(big.Int)

			for i := 0; i < 16; i++ {
				for j := 0; j < 8; j++ {
					bigInt.SetBit(bigInt, i*8+j, uint((b[i]&(0b1<<j))>>j))
				}
			}
			return bigInt.String(), 16
		}
	case "u64":
		if len(data[offset:]) < 8 {
			return nil, 8
		} else {
			return binary.LittleEndian.Uint64(data[offset : offset+8]), 8
		}
	case "u32":
		if len(data[offset:]) < 4 {
			return nil, 4
		} else {
			return binary.LittleEndian.Uint32(data[offset : offset+4]), 4
		}
	case "u16":
		if len(data[offset:]) < 2 {
			return nil, 2
		} else {
			return binary.LittleEndian.Uint16(data[offset : offset+2]), 2
		}
	case "u8":
		if len(data[offset:]) < 1 {
			return nil, 1
		} else {
			return data[offset], 1
		}

		// that particular binary number uses Two's complement bit weight
		// https://en.wikipedia.org/wiki/Sign_bit#Sign_bit_weight_in_Two's_Complement
		// so if the number is negative all bits should be reversed
		// and later we should add one and then negate the whole thing
		// lets say we have 1001, to get the number we should
		// - check the most significant bit
		// - if it is 0 then procceed like a regular binary number
		// - if it is 1:
		// 1) reverse bits 1001 => 0110
		// 2) add one 0110 => 0111
		// 3) negate 0111 => -0111
		// and we got -7, so 1001 is -7
	case "i128":
		if len(data[offset:]) < 16 {
			return nil, 16
		} else {
			b := data[offset : offset+16]
			bigInt := new(big.Int)

			// get last bit:
			// this number uses little-endian, it means that last byte is 15-th
			// to get last bit of that byte we should do bitwise AND
			// <lastByte> AND 10000000 => <lastByte> AND 1 << 7 => <lastByte> AND 0b1 << 7
			// but in that way we will get either '10000000' or '0000000' and we want 1 or 0
			// so we need to shift that bit so it becomes first (7 times)
			// so we will get: (<lastByte> AND 0b1<<7)>>7
			sign := uint8(int((b[15] & (0b1 << 7)) >> 7))
			for i := 0; i < 16; i++ {
				for j := 0; j < 8; j++ {

					// here we take j-th bit and do xor with sign
					// so if sign is 1 it means that number is negative
					// and if number is negative we should negate all the bits
					// sign|bit before|bit after| xor |
					//  0  |     0    |    0    |0^0=0|
					//  0  |     1    |    1    |0^1=1|
					//  1  |     0    |    1    |1^0=1|
					//  1  |     1    |    0    |1^1=0|
					// as you can see if sign is one, bit gets negated,
					// and if sign is zero, it gets untouched
					bigInt.SetBit(bigInt, i*8+j, uint(((b[i]&(0b1<<j))>>j)^sign))
				}
			}

			bigInt.Add(bigInt, big.NewInt(int64(sign)))
			if sign == 1 {
				bigInt.Neg(bigInt)
			}

			return bigInt.String(), 16
		}
	case "i64":
		if len(data[offset:]) < 8 {
			return nil, 8
		} else {
			b := data[offset : offset+8]

			num := int64(binary.LittleEndian.Uint64(b))
			return num, 8
		}
	case "i32":
		if len(data[offset:]) < 4 {
			return nil, 4
		} else {
			b := data[offset : offset+4]

			num := int32(binary.LittleEndian.Uint32(b))
			return num, 4
		}
	case "i16":
		if len(data[offset:]) < 2 {
			return nil, 2
		} else {
			b := data[offset : offset+2]

			num := int16(binary.LittleEndian.Uint16(b))
			return num, 2
		}
	case "i8":
		if len(data[offset:]) < 1 {
			return nil, 1
		} else {
			ub := data[offset]

			b := int8(ub)

			return b, 1
		}
	case "f64":
		if len(data[offset:]) < 8 {
			return nil, 8
		} else {
			a := binary.LittleEndian.Uint64(data[offset : offset+8])
			f64 := math.Float64frombits(a)
			return f64, 8
		}
	case "f32":
		if len(data[offset:]) < 4 {
			return nil, 4
		} else {
			a := binary.LittleEndian.Uint32(data[offset : offset+4])
			f32 := math.Float32frombits(a)
			return f32, 4
		}
	case "bool":
		if len(data[offset:]) < 1 {
			return nil, 1
		} else {
			return bool(data[offset]&0b00000001 == 1), 1
		}
	case "publicKey":
		if len(data[offset:]) < 32 {
			return nil, 32
		} else {
			return base58.Encode(data[offset : offset+32]), 32
		}
	case "pubkey":
		if len(data[offset:]) < 32 {
			return nil, 32
		} else {
			return base58.Encode(data[offset : offset+32]), 32
		}
	case "string":
		strLen := binary.LittleEndian.Uint32(data[offset : offset+4])
		var n int = 4
		if len(data[offset+n:]) < int(strLen) {
			return nil, n
		} else {
			return string(data[offset+n : offset+n+int(strLen)]), n + int(strLen)
		}
	case "bytes":
		return extractVector(data, nil, offset, "u8")
	}
	return nil, 0
}
