//go:build fips_enabled

package main

import (
	_ "crypto/tls/fipsonly"
	"fmt"
)

func init() {
	fmt.Println("***** Starting with FIPS crypto enabled *****")
}
