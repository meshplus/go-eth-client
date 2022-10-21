pragma solidity ^0.8.4;

import "./contracts/token/ERC721/ERC721.sol";

contract PandaNft is ERC721{

    uint public MAX_APES = 10000; // 总量
    uint count = 0;
    address public owner;
    string private _tokenBaseURI;


    constructor(string memory name_, string memory symbol_) ERC721(name_, symbol_){
        owner =msg.sender;
    }

    function setBaseURI(string memory newBaseURI) external {
        require(msg.sender==owner,"only contract owner can set baseUri");
        _tokenBaseURI=newBaseURI;
    }

    function getBaseURI() public view returns (string memory) {
        return _baseURI();
    }


    function _baseURI() internal view override returns (string memory) {
        return _tokenBaseURI;
    }

    function mint(address to, uint tokenId) external {
        //require(msg.sender==owner,"only contract owner can mint token");
        require(tokenId >= 0 && tokenId < MAX_APES, "tokenId out of range");
        _mint(to, tokenId);
        count++;
    }

    function tokensOfOwnerIn(
        address owner
    ) external view virtual returns (uint256[] memory) {
        uint256 tokenIdsIdx;
        uint256 tokenIdsMaxLength = balanceOf(owner);
        uint256[] memory tokenIds = new uint256[](tokenIdsMaxLength);
        for(uint256 i=1;i<=count&&tokenIdsIdx!=tokenIdsMaxLength;i++){
            if (ownerOf(i)== owner) {
                tokenIds[tokenIdsIdx++] = i;
            }
        }
        return tokenIds;

    }

    function getCount() public view returns (uint) {
        return count;
    }




}