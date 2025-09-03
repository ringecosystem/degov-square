package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

const otpLength = 6

// nodeId is used to identify the node in a distributed system, ensuring that IDs generated from different nodes do not collide.
func InitIDGenerator(nodeID int64) error {
	var err error
	once.Do(func() {
		node, err = snowflake.NewNode(nodeID)
	})
	return err
}

// generate next unique ID
func NextID() int64 {
	if node == nil {
		// if not initialized, use default node ID 0
		_ = InitIDGenerator(1)
	}
	return node.Generate().Int64()
}

// nextIDString generates the next unique ID in string format
func NextIDString() string {
	if node == nil {
		_ = InitIDGenerator(1)
	}
	return node.Generate().String()
}

func NextOTPCode() (string, error) {
	max := new(big.Int)
	max.Exp(big.NewInt(10), big.NewInt(int64(otpLength)), nil)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("error generating OTP code: %w", err)
	}

	format := fmt.Sprintf("%%0%dd", otpLength)
	otpCode := fmt.Sprintf(format, n)

	return otpCode, nil
}
