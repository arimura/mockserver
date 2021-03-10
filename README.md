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
## Redirection
`data/redirec` is special path for redirection.
```sh
$ cat data/redirect/fuga
https://example.com

$ curl http://localhost:8000/redirect/fuga -v
*   Trying ::1...
* TCP_NODELAY set
* Connected to localhost (::1) port 8000 (#0)
> GET /redirect/test HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 301 Moved Permanently
< Location: https://example.com
< Date: Wed, 10 Mar 2021 08:31:51 GMT
< Content-Length: 0
```
