# Kiesnet Token Chaincode

An account/balance based token

#

## requirement
- kiesnet-id chaincode (devmode: kiesnet-cc-id)
- kiesnet-contract chaincode (devmode: kiesnet-cc-contract)
- knt-{_token_code_} chaincodes (devmode: knt-cc-{_token_code_})

#

## Terms

- PAOT : personal(main) account of the token

#

## API

method __`func`__ [arg1, _arg2_, ... ] {trs1, _trs2_, ... }
- method : __query__ or __invoke__
- func : function name
- [arg] : mandatory argument
- [_arg_] : optional argument
- {trs} : mandatory transient
- {_trs_} : optional transient

#

> invoke __`account/create`__ [token_code, _co-holders..._] {_"kiesnet-id/pin"_}
- Create an account
- [token_code] : issued token code
- [_co-holders..._] : PAOTs (exclude invoker, max 127)
- If holders(include invoker) are more then 1, it creates a joint account. If not, it creates the PAOT.

> query __`account/get`__ [token_code|address]
- Get the account
- If the parameter is token code, it returns the PAOT.
- account types
    - 0x00 : unknown
    - 0x01 : personal
    - 0x02 : joint

> invoke __`account/holder/add`__ [account, holder] {_"kiesnet-id/pin"_}
- Create a contract to add the holder
- [account] : the joint account address
- [holder] : PAOT of the holder to be added

> invoke __`account/holder/remove`__ [account, holder] {_"kiesnet-id/pin"_}
- Create a contract to remove the holder
- [account] : the joint account address
- [holder] : PAOT of the holder to be removed

> query __`account/list`__ [token_code, _bookmark_, _fetch_size_]
- Get account list
- If token_code is empty, it returns all account regardless of tokens.
- [_fetch_size_] : max 200, if it is less than 1, default size will be used (20)

> invoke __`account/suspend`__ [token_code] {_"kiesnet-id/pin"_}
- Suspend the PAOT

> invoke __`account/unsuspend`__ [token_code] {_"kiesnet-id/pin"_}
- Unsuspend the PAOT

> query __`balance/logs`__ [token_code|address, _bookmark_, _fetch_size_, _starttime_, _endtime_]
- Get balance logs
- If the parameter is token code, it returns logs of the PAOT.
- [_fetch_size_] : max 200, if it is less than 1, default size will be used (20)
- [_starttime_] : __time(seconds)__ represented by int64
- [_endtime_] : __time(seconds)__ represented by int64
- log types
    - 0x00 : mint
    - 0x01 : burn
    - 0x02 : send
    - 0x03 : receive
    - 0x04 : deposit (create a pending balance)
    - 0x05 : withdraw (from the pending balance)

> query __`balance/pending/list`__ [token_code|address, _sort_, _bookmark_, _fetch_size_]
- Get pending balances list
- If the parameter is token code, it returns logs of the PAOT.
- [_sort_] : 'created_time' or 'pending_time'(default)
- [_fetch_size_] : max 200, if it is less than 1, default size will be used (20)
- pending types
    - 0x00 : account
    - 0x01 : contract

> invoke __`balance/pending/withdraw`__ [pending_balance_id] {_"kiesnet-id/pin"_}
- Withdraw the balance

> invoke __`token/burn`__ [token_code, amount] {_"kiesnet-id/pin"_}
- Get the burnable amount and burn the amount.
- [amount] : big int
- If genesis account holders are more than 1, it creates a contract.

> invoke __`token/create`__ [token_code, _co-holders..._] {_"kiesnet-id/pin"_}
- Create(Issue) the token
- [token_code] : 3~6 alphanum
- [_co-holders..._] : PAOTs (exclude invoker, max 127)
- It queries meta-data of the token from the knt-{token_code} chaincode.

> query __`token/get`__ [token_code]
- Get the current state of the token

> invoke __`token/mint`__ [token_code, amount] {_"kiesnet-id/pin"_}
- Get the mintable amount and mint the amount.
- [amount] : big int
- If genesis account holders are more than 1, it creates a contract.

> invoke __`transfer`__ [sender, receiver, amount, _memo_, _pending_time_, _expiry_, _extra-signers..._] {_"kiesnet-id/pin"_}
- Transfer the amount of the token or create a contract
- [sender] : an account address, __empty = PAOT__
- [receiver] : an account address
- [amount] : big int
- [_memo_] : max 128 charactors
- [_pending_time_] : __time(seconds)__ represented by int64
- [_expiry_] : __duration(seconds)__ represented by int64, multi-sig only
- [_extra-signers..._] : PAOTs (exclude invoker, max 127)

> query __`ver`__
- Get version
