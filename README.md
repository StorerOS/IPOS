# IPOS

IPOS is an open source decentralized storage protocol that provides [AWS S3](https://aws.amazon.com/cn/s3/) native APIs for developers. IPOS will distribute data stored based on the AWS S3 standard in IPFS. It supports IPFS mainet and  private networks by the deployment of IPFS nodes. So as to meet the distributed storage needs of developers.

## Get Start

### Open [IPOS Website](http://14.215.91.114:8082/ipos/)

![image](https://user-images.githubusercontent.com/90947287/134447556-b1565a91-1417-410f-a593-48846f47f1bc.png)

1. Enter AccessKey and SecretKey

test key
```
AccessKey: iposadmin
SecretKey: iposadmin
```

![image](https://user-images.githubusercontent.com/90947287/134448622-54c59154-69c0-458a-89b4-b4140098d918.png)


2. Create bucket and Upload file

![image](https://user-images.githubusercontent.com/90947287/134448711-b9511cf8-ac2f-4fa0-827f-f20f838a9883.png)


3. Create bucket and Upload file success

![image](https://user-images.githubusercontent.com/90947287/134448864-a6e8120f-a19a-40f9-907f-190b83ab8ebe.png)

The file(image.png) is stored in IPFS which is the private network now. You can share,preview,delete it and so on.


## Build

```
make build
```

## Run

```
build/ipos server
```
