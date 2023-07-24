## router
合约仓库： https://github.com/deltaswapio/swaprouter/tree/feature/near

## near
合约仓库： https://github.com/deltaswapio/near-contract 

常用api文档：https://docs.near.org/docs/api/overview

> testnet  
rpc:  https://archival-rpc.testnet.near.org
chain_id:  1001313161555  
> mainnet  
rpc: https://archival-rpc.mainnet.near.org
chain_id: 1001313161554

## router部署文档 
https://github.com/deltaswapio/swaprouter/tree/feature/near#readme
## mpc部署文档 
https://github.com/deltaswapio/FastMulThreshold-DSA/wiki/keygen-and-sign-workflow

> 交易参考(bsc->near)  

***
特别强调  
```text
>1) mpc公钥和near公钥的关系  
mpc申请ed公钥(32字节16进制编码字符串)后，公钥本身就是一个near的account，也可以添加公钥到特定账户，转入一笔初始金额后，即激活  
另外，mpc获取的公钥，通过  https://github.com/deltaswapio/swaprouter/blob/feature/near/tokens/near/tools/publicKeyToAddress/main.go  工具可获得near publicKey  
示例：  
go run tokens/near/tools/publicKeyToAddress/main.go f353e1fe460864caf4d720e40e57f14d35f437c3e0b93d1f40a37e89ebdda3bf
INFO[2022-05-08T09:58:16.178] convert public key to address success        
INFO[2022-05-08T09:58:16.178] nearAddress is f353e1fe460864caf4d720e40e57f14d35f437c3e0b93d1f40a37e89ebdda3bf 
INFO[2022-05-08T09:58:16.178] nearPublicKey is ed25519:HNrFuGeXk7WGXkX2BhRzVK2B7a9E6HLGSujF1uHZAvNa
```
```text
>2) nep141和erc20的关系  
nep141是near上的同质化代币协议，即near上的erc20  
主要区别有以下几点:  
① nep141协议没有approve和transfer_from，所以只能通过ft_transfer_call发送到合约，合约做逻辑处理（也是我们的跨链处理逻辑）  
② nep141协议规定，所有接收代币的账户必须注册抵押，即storage_deposit方法质押0.0025个near在合约上，才能持有该合约的代币  
③ nep141的transfer有两种，ft_transfer转账，ft_transfer_call(接收方只能是合约)转账的同时，接收账户做逻辑处理  
④ nep141规定所有的transfer方法都必须支付1个yocto（1near=1*10**24yocto），即--depositYocto 1
```
```text
>3) router config  
anytoken: contractVersion=666
native: contractVersion=999
```
```text
>4) deploy mpcPool
go run ./tokens/near/tools/deployContract/main.go -config ./build/bin/config-sign-with-privatekey-example.toml -chainID 1001313161555 -pubKey ed25519:7SVZCtsvrQmmAk9q5Ds4eZxKHWpgkQTSwNud5kn9JLiK -privKey ed25519:5NNdYaMoxpKZNTft2vrfx11tt9Lk5W7Zo3dkJkGRmZboEEHYEiJUzowdMWqTXSgfMKQcWNmD17zTdXrViRCsmTmH -accountId test.userdemo.testnet
```
***

## mpc地址账户创建步骤
```text
>1) 调用go run tokens/near/tools/publicKeyToAddress/main.go 获取mpc对应的near公钥
```
```text
>2) 调用go run ./tokens/near/tools/functionCall/main.go -config xx.toml -chainID xx -network [testnet/near] -functionName create_account -pubKey [signerPublicKey] -privKey  [signerPrivKey] -newAccountId [newMpcAccountId] -newPublicKey [mpcNearPublicKey] -amount [initBalance] -accountId [signerAccountId]
```
```text
>3) newMpcAccountId即为mpc账户的accountId
```

## near部署步骤
```text
>1)安装rust环境
安装rust:  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh  
添加工具链: rustup target add wasm32-unknown-unknown  
//安装完成后，运行cargo version查看是否成功安装
```

```text
>2)安装near交互工具 
# near没有ui交互工具,可能是我没发现
npm install -g near-cli  
//安装完成后，运行near --version查看是否成功安装
``` 

```text
>3)登录near-cli
near login （~/.near-credentials文件夹下生成密钥对文件）
或者 near generate-key ACCOUNT_ID --seedPhrase="xxx"
```

```text
>4)创建子账户  
# 主账户才有权限创建名下子账户，用户部署合约
near create-account CONTRACT_NAME.ACCOUNT_ID --masterAccount ACCOUNT_ID --initialBalance 10
```

```text
>5)near合约部署(anytoken/nep141合约同理)  
# 进入对应的根目录
cd near-contract/router
# 编译合约，编译后，在target/wasm32-unknown-unknown/release下有router.wasm文件
env 'RUSTFLAGS=-C link-arg=-s' cargo build --target wasm32-unknown-unknown --release 
# 部署合约到指定账户
near deploy --wasmFile *.wasm --accountId CONTRACT_ID
```

```text
>6)初始化合约  
# nep141合约
near call nep141.CONTRACT_ID new_default_meta '{"owner_id":"xxxx","total_supply":"xxxxx"}' --accountId ACCOUNT_ID 
```

```text
>7)注册存储
# 接收代币的账户都需要注册存储
near call nep141.CONTRACT_ID storage_deposit '{"account_id":"xxx"}' --accountId  ACCOUNT_ID  --deposit 1
```

```text
>8)跨出交易发起
near call nep141.CONTRACT_ID ft_transfer '{"receiver_id": "mpc","amount": "xxx","memo": "bindaddr tochainId"}' --accountId ACCOUNT_ID --gas 300000000000000 --depositYocto 1
```

## 常见问题
```text
>1)linker `cc` not found  
sudo apt install build-essential
```
```text
>2)near command not found  
# 查询全局路径
node config ls 查询全局路径
# 配置环境变量
export PATH="全局路径:$PATH"
```


  
