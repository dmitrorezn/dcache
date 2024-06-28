pragma solidity ^0.8.24;

import "https://github.com/OpenZeppelin/openzeppelin-contracts/blob/v3.0.0/contracts/token/ERC20/IERC20.sol"

contract Game {
    event BetDone(address addr, uint amount);
    event Won(address addr, uint amount);
    //event Deposited(address addr, uint amount);
    //event Withdrowed(address addr, uint amount);

    mapping(address => uint) public balances;
    address public gameAddress;

    constructor(address memory _gameAddress) public {
        gameAddress = _gameAddress;
    }

    function GetBalance(
        address addr
    ) public view returns (uint balance)
    {
        return balances[addr];
    }
    function Deposit(
        address addr,
        ERC20 token,
        uint amount
    )  public {
        require(amount>0);
        token.transferFrom(address(this), gameAddress, amount);
        balances[addr]+=amount;
    }
    function Withdraw(
        address addr,
        ERC20 token,
        uint amount
    ) public {
        require(balances[addr]>=amount);
        balances[addr]-=amount;

        token.transferFrom(gameAddress, address(this), amount);
    }

    function MakeBet(address addr, uint amount) public {
        require(balances[addr]>=amount);

        balances[addr]-=amount;

        emit BetDone(addr, amount);
    }

    function Win(address addr, uint amount) public {
        require(amount>0);

        balances[addr]+=amount;

        emit Won(addr, amount);
    }
}