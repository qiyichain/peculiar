// SPDX-License-Identifier: MIT

pragma solidity ^0.8.15;

import "@openzeppelin/contracts/token/ERC721/extensions/ERC721URIStorage.sol";
import "@openzeppelin/contracts/utils/Counters.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract PeculiarNFT is ERC721URIStorage, Ownable {
    using Counters for Counters.Counter;
    Counters.Counter private _tokenIds;
    
    // storage
    Counters.Counter public _mintIds;
    Counters.Counter public _transferIds;
    
    constructor() ERC721("PeculiarNFT", "PEC") {}

    function Mint(address to, string memory tokenURI)
        public
        onlyOwner
        returns (uint256)
    {
        uint256 newItemId = _tokenIds.current();
        _mint(to, newItemId);
        _setTokenURI(newItemId, tokenURI);

        _tokenIds.increment();
        _mintIds.increment();
        return newItemId;
    }

    function TransferToken(address to, uint256 tokenId) public {
        _transfer(msg.sender, to, tokenId);
        _transferIds.increment();
    }

    function CurrentId() public view returns(uint256)  {
        return _tokenIds.current();
    }

    function BatchTrans(address to, uint256 num) public onlyOwner {
        for (uint256 i = 0; i < num; i++) {
            Mint(to, "http://metaverse.org/token/id/");
        }
    }

    function Reset() public onlyOwner {
        _mintIds.reset();
        _transferIds.reset();
    }

}
