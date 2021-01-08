# mockserver

Tiny mock server for you

## How to use
```sh
$ echo "hoge" > data/hoge
$ make run
$ curl 'http://localhost:8000/hoge'
```

## Macro
In macroExpand mode, JSON value of POST body can be referred by response data.
```sh
$ cat data/hoge
Hello {{ .foo}}

$ curl 'http://localhost:8000/hoge' -d '{"foo":"bar"}'
Hello bar
```