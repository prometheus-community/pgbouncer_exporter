# How to run tests

Assuming you are in test folder

*odyssey:*
```
$ cd odyssey/
$ docker-compose up --build
$ ../compare.sh odyssey/expected.out
```

*pgbouncer:*
```
$ cd pgbouncer
$ docker-compose up --build
$ ./compare.sh pgbouncer/expected.out
```
