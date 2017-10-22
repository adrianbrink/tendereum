# tendereum

[![CircleCI](https://circleci.com/gh/adrianbrink/tendereum/tree/master.svg?style=svg)](https://circleci.com/gh/adrianbrink/tendereum/tree/master)

## Design

```
                =============================================
============    =  ===============         ===============  =       
=          =    =  = RPC Server  =         = State App   =  =
=  Web3    <---------->          = <------ =             =  =
=  RPC     =    =  =             =         =             =  =
=  Client  =    =  =             =         =             =  =
=          =    =  = -API        =         = -EVM        =  =
============    =  = -Acc Mann   =         = -Trie       =  =
                =  =             =         = -Database   =  =
                =  ===============         ===============  =
                =             |                   ^         =
                ==============|===================|==========
                              |Txs                |Txs
                ==============|===================|==========
                = Platform    |                   |         =
                =             |            ===============  =                                             
                =             |            = TMSP Server =  = 
                =             |            ===============  =
                =             |            ^      ^      ^  =
                =             |            |      |      |  =
                =             |       TMSP |      |      |  =
                =             |            |      |      |  =
                =             v            v      v      v  =
                = ========================================= =  
                = = Tendermint Core                       = =
                = ========================================= =  
                =                   ^                       =
                ====================|========================  
                                    |
                                    |
                                    v
                                Consensus
```
