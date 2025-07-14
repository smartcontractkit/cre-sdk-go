package bindings

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func NewBalanceReaderAbi() abi.ABI {
	// Parse the ABI from the raw string
	parsedAbi, _ := abi.JSON(strings.NewReader(balanceReaderABI))
	return parsedAbi
}

func NewIERC20Abi() abi.ABI {
	// Parse the ABI from the raw string
	parsedAbi, _ := abi.JSON(strings.NewReader(iErc20Raw))
	return parsedAbi
}

func NewIReserveManagerAbi() abi.ABI {
	a, _ := abi.JSON(strings.NewReader(iReserveManagerRaw))
	return a
}

func NewMessageEmitterAbi() abi.ABI {
	a, _ := abi.JSON(strings.NewReader(messageEmitterABI))
	return a
}
