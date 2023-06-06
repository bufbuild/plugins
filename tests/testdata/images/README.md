# buf images

## Description

For reproducable testing, we export images using `buf build` from the BSR and store them here.

### eliza.bin.gz

This is the https://buf.build/bufbuild/eliza/docs/dbde79169a014bd8b5bf8f89ac9b35c7 commit of bufbuild/eliza, exported with:

```
$ buf build buf.build/bufbuild/eliza:dbde79169a014bd8b5bf8f89ac9b35c7 -o eliza.bin.gz
```

### petapis.bin.gz

This is the https://buf.build/acme/petapis/commits/84a33a06f0954823a6f2a089fb1bb82e commit of acme/petapis, exported with:

```
$ buf build buf.build/acme/petapis:84a33a06f0954823a6f2a089fb1bb82e -o petapis.bin.gz
```

### grpc-gateway.bin.gz

This is the https://github.com/grpc-ecosystem/grpc-gateway/releases/tag/v2.15.2 commit of grpc-ecosystem/grpc-gateway, exported with

```
$ buf build bufbuild.internal/grpc-ecosystem/grpc-gateway/commits:{DYNAMIC-BSR-COMMIT} -o grpc-gateway.bin.gz
```

The content must first be manually pushed to the local BSR instance from the grpc-gateway repository root by adjusting
the host, logging into the registry, updating then pushing the module.

The BSR-hosted version of the repository is annotations only and does not produce valuable test output.
