
# ipfs-multigateway

<a href="https://travis-ci.org/schollz/ipfs-multigateway"><img src="https://img.shields.io/travis/schollz/ipfs-multigateway.svg?style=flat-square" alt="Build Status"></a>

This enables a local server that will request a single hash from as many IPFS Gateways as possible, returning the first result found and canceling all the other requests before they finish. This way you can request `/ipfs/<hash>` as fast as possible.

## Install

```
go get github.com/schollz/ipfs-multigateway
```

## Usage

```
$ ipfs-multigateway
[info]  2019/05/09 07:05:01 checking gateways...
[info]  2019/05/09 07:05:06 found 22 functional gateways                  
[info]  2019/05/09 07:05:06 running on :8085
```

Now you try opening up a IPFS hash like `localhost:8085/ipfs/QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG/readme`.

## Contributing

Pull requests are welcome. Feel free to...

- Revise documentation
- Add new features
- Fix bugs
- Suggest improvements

## License

MIT
