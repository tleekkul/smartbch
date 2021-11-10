// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

contract Contract1 {

    uint256 public counter;
    address public contract2;
    address public contract3;

    constructor(address _contract2, address _contract3) {
        contract2 = _contract2;
        contract3 = _contract3;
    }

    function call2(uint256 n) public returns (uint256) {
        counter += n * 100;
        Contract2(contract2).call3(n + 1);
        return counter;
    }
    function call3(uint256 n) public returns (uint256) {
        counter += n * 100;
        Contract3(contract3).callMe(n + 1);
        return counter;
    }

}

contract Contract2 {

    uint256 public counter;
    address public contract3;

    constructor(address _contract3) {
        contract3 = _contract3;
    }

    function call3(uint256 n) public returns (uint256) {
        counter += n * 200;
        Contract3(contract3).callMe(n + 1);
        return counter;
    }

}

contract Contract3 {

    uint256 public counter;

    function callMe(uint256 n) public returns (uint256) {
        counter += n * 300;
        return counter;
    }

}
