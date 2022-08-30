// SPDX-License-Identifier: MIT

pragma solidity ^0.8.15;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/utils/Counters.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract PeculiarToken is ERC20, Ownable {
    // storage
    using Counters for Counters.Counter;
    Counters.Counter public _mintIds;
    Counters.Counter public _transferIds;


    constructor() ERC20("ERC20_Token", "ERC") {
        uint256 initialSupply = 1e18 * 1e8;
        _mint(msg.sender, initialSupply);
    }
    
    function mint(address to, uint256 supply)
        public
        onlyOwner
    {
        _mint(to, supply);
        _mintIds.increment();
    }

    function transfer(address to, uint256 amount) public virtual override returns (bool) {
        address owner = _msgSender();
        _transfer(owner, to, amount);
        _transferIds.increment();
        return true;
    }

    function Reset() public onlyOwner {
        _mintIds.reset();
        _transferIds.reset();
    }

}
