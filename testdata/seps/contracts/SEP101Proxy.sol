// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import "./interfaces/SEP101.sol";

contract SEP101Proxy is SEP101 {

    bytes4 private constant _SELECTOR_SET = bytes4(keccak256(bytes("set(bytes,bytes)")));
    bytes4 private constant _SELECTOR_GET = bytes4(keccak256(bytes("get(bytes)")));

    address constant public agent = address(0x2712);

    function set(bytes calldata key, bytes calldata value) override external {
        agent.call(abi.encodeWithSelector(_SELECTOR_SET, key, value));
    }
    function get(bytes calldata key) override external view returns (bytes memory) {
        (bool success, bytes memory data) = agent.staticcall(abi.encodeWithSelector(_SELECTOR_GET, key));
        return abi.decode(data, (bytes));
    }

}
