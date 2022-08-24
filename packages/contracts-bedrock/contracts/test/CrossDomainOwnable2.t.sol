// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest, Messenger_Initializer } from "./CommonTest.t.sol";
import { CrossDomainOwnable2 } from "../L2/CrossDomainOwnable2.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { Vm } from "forge-std/Vm.sol";
import { Bytes32AddressLib } from "@rari-capital/solmate/src/utils/Bytes32AddressLib.sol";
import { Hashing } from "../libraries/Hashing.sol";

contract XDomainSetter2 is CrossDomainOwnable2 {
    uint256 public value;

    function set(uint256 _value) external onlyOwner {
        value = _value;
    }
}

contract CrossDomainOwnable2_Test is Messenger_Initializer {
    XDomainSetter2 setter;

    function setUp() override public {
        super.setUp();
        vm.prank(alice);
        setter = new XDomainSetter2();
    }

    function test_revertNotSetOnlyOwner() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        setter.set(1);
    }

    function test_revertOnlyOwner() external {
        uint256 nonce = 0;
        address sender = AddressAliasHelper.applyL1ToL2Alias(bob);
        address target = address(setter);
        uint256 value = 0;
        uint256 minGasLimit = 0;
        bytes memory message = abi.encodeWithSelector(
            XDomainSetter2.set.selector,
            1
        );

        bytes32 hash = Hashing.hashCrossDomainMessage(
            nonce,
            sender,
            target,
            value,
            minGasLimit,
            message
        );

        // It should be a failed message. The revert is caught,
        // so we cannot expectRevert here.
        vm.expectEmit(true, true, true, true);
        emit FailedRelayedMessage(hash);

        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)));
        L2Messenger.relayMessage(
            nonce,
            sender,
            target,
            value,
            minGasLimit,
            message
        );

        assertEq(
            setter.value(),
            0
        );
    }

    function test_onlyOwner() external {
        address owner = setter.owner();

        // Simulate the L2 execution where the call is coming from
        // the L1CrossDomainMessenger
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)));
        L2Messenger.relayMessage(
            1,
            AddressAliasHelper.applyL1ToL2Alias(owner),
            address(setter),
            0,
            0,
            abi.encodeWithSelector(
                XDomainSetter2.set.selector,
                2
            )
        );

        assertEq(
            setter.value(),
            2
        );
    }
}
