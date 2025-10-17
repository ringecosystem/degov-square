package utils

import "strings"

func IsNativeToken(address string) bool {
	lowerCaseAddress := strings.ToLower(address)
	isNative := lowerCaseAddress == "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee" || lowerCaseAddress == "0x0000000000000000000000000000000000000000"
	return isNative
}
