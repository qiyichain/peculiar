// SPDX-License-Identifier: MIT

pragma solidity ^0.8.15;

import "@openzeppelin/contracts/token/ERC1155/presets/ERC1155PresetMinterPauser.sol";

contract GamePECLR is ERC1155PresetMinterPauser {
    uint256 public constant CHALLENGER = 0;
    uint256 public constant DIAMOND = 1;
    uint256 public constant PLATNUM = 2;
    uint256 public constant GOLD = 3;
    uint256 public constant SILVER = 4;

    constructor() ERC1155PresetMinterPauser("https://game.peclr.io/api/nft/{id}.json") {
        _mint(msg.sender, CHALLENGER, 10**18, "");
        _mint(msg.sender, DIAMOND, 10**27, "");
        _mint(msg.sender, PLATNUM, 1, "");
        _mint(msg.sender, GOLD, 10**9, "");
        _mint(msg.sender, SILVER, 10**9, "");
    }
}