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