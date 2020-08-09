# spanner-golang-test
A golang unit test example for google spanner, using google spanner emulator provided by [cloud-spanner-emulator](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator).

Hopefully, it helps you.
I create this since I can not find the straight-forward example for golang.

# Setup

You need spanner emulator running:

```sh
docker run -d -p 9010:9010 -p 9020:9020 gcr.io/cloud-spanner-emulator/emulator:1.0.0
```

# Test

Run `make test`. The test uses golang test suite, which will do:
1. Create spanner instance on google spanner emulator
2. Create spanner db on google spanner emulator
3. Execute [singer-table-ddl](internal/ddl/spanner/001_create_singer_table.sql)
4. Truncate `singer` table before test
5. Execute test(s)
6. Drop database and instance on google spanner emulator

Database creation/drop is done as suggested by in [cloud-spanner-emulator](https://github.com/GoogleCloudPlatform/cloud-spanner-emulator) README.


# Quick Links

Parts of code that might help you:
- [singer_test](internal/singer/spanner/singer_test.go)
- [github action example](.github/workflows/test.yml)
- [gitlab ci sample](.gitlab-ci.yml.sample)
