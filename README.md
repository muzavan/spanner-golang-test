# spanner-golang-test
A golang unit test example for google spanner, using [cloud-spanner-emulator](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator).

Hopefully, it helps you.
I create this since I can not find the straight-forward example for golang.

# Setup

You need spanner emulator running:

```sh
docker run -p 9010:9010 -p 9020:9020 gcr.io/cloud-spanner-emulator/emulator:1.0.0
```
