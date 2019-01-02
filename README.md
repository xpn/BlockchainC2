# BlockchainC2

BlockchainC2 is a small POC server/agent to assess how the Blockchain (specifically Ethereum's Smart Contract functionality) can be used by an attacker for C2.

Details regarding this application can be found here.

## Smart Contract

The Smart Contract used within this POC is quite simple:

```
pragma solidity ^0.5.0;

contract EventC2 {

    address owner;

    event _ServerData(bool f, bool enc, int seq, string agentID, string data);
    event _ClientData(bool f, bool enc, int seq, string agentID, string data);

    constructor() public {
        owner = msg.sender;
    }
    
    function AddClientData(string memory agentID, string memory d, int id, bool f, bool enc) public {
        emit _ClientData(f, enc, id, agentID, d);
    }

    function AddServerData(string memory agentID, string memory d, int id, bool f, bool enc) public {
        emit _ServerData(f, enc, id, agentID, d);
    }
}
```

The focus of this Solidity code is to pass events between a server and multiple clients in the form of events. 

## Building

BlockchainC2 was designed to be run on MacOS/Linux, but the agent can be compiled to execute on MacOS, Windows, or Linux.

To build on MacOS using brew:

```
# Install solidity and ethereum
brew tap ethereum/ethereum
brew install ethereum
brew install solidity

# Build
make all
```

To build on Ubuntu:

```
# Install solidity and ethereum
sudo add-apt-repository ppa:ethereum/ethereum
sudo apt-get update
sudo apt-get install solc ethereum

# Build  
make all
```

To cross-compile an agent for Windows:

```
CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" GOOS=windows go build blockchainc2/cmd/bc2agent
```

## Running

You will need to setup an account which can be used by the server component. The easiest way to do this is with `geth`:

```
geth account new --keystore /tmp/mykeystore/
cat /tmp/mykeystore/*
```

Ether can be added to your wallet on the Ropsten testnet using https://faucet.ropsten.be/.

Add the keychain to your config.json, for example:

```
{
	"Key": "{\"address\":\"ADDRESS\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"CT\",\"cipherparams\":{\"iv\":\"IV\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":262144,\"p\":1,\"r\":8,\"salt\":\"06470fcc2121994e014f85e5ab9cdb3714c76b873a1f1186c3e623e87abc4a7a\"},\"mac\":\"SALT\"},\"id\":\"ID\",\"version\":3}",
	"Endpoint": "wss://ropsten.infura.io/_ws",
	"ContractAddress": "TODO_VIA_SETUP",
	"GasPrice": 0
}
```

To deploy a contract using `bc2server`:

```
./bin/bc2server -config ./config.json -pass Passw0rd -setup
```

Once the contract has been deployed, add the address to your config.json and start the server with:

```
./bin/bc2server -config ./config.json -pass Passw0rd
```

With the server running, agents can be attached using:

```
./bin/bc2agent -config ./agent_config.json -pass Passw0rd
```

It is recommended that a fresh account is used for an agent to avoid errors with pending transactions.