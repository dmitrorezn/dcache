generate: prepare
	- solc --abi contracts/game.sol > contracts/game.abi
	- solc --bin contracts/game.sol > contracts/game.bin
	- abigen --bin=contracts/game.bin --abi=contracts/game.abi --pkg=game --out=contracts/game.go

prepare:
	- eval $(/opt/homebrew/bin/brew shellenv)
