# xtop
A top-like tool to monitor HTTP responses from a target URL

### Installation
```
$ go get github.com/solidfoxrock/xtop
```

### Usage

```
$ xtop --help
usage: xtop [<flags>] <url>

A top-like tool to monitor responses from a target URL. This tool periodically collects and prints out response statuses and a custom
response header received from the url.

Flags:
  --help     Show help.
  --version  Show application version.
  -c, --concurrency=10
             max number of concurrent requests
  -x, --header="X-Server"
             custom header name to collect

Args:
  <url>  target URL
```

### Example

```
$ xtop -c 3 -x X-Served-By github.com
```

```
    Target: http://github.com
    Header: X-Served-By
    Max concurrency: 3

    === Response status ===
    [100%]  [72/72] 200 OK

    === Response header X-Served-By ===
     [5%]   [4/72]   1 1868c9f28a71d80b2987f48dbd1824a0
     [1%]   [1/72]   2 1c0ce1a213af16e49d5419559ef44f50
     [9%]   [7/72]   3 353c3e3783cd5fa717715b8bdf83c23b
     [4%]   [3/72]   4 362482c1f05726391203e2d2c32818a4
     [5%]   [4/72]   5 3f38dada85f97412f7f824e59f77fa9d
     [1%]   [1/72]   6 44f77bef9757b092723b0a6870733b02
     [4%]   [3/72]   7 4580e595d77515caa5593194518fa90f
     [5%]   [4/72]   8 50f1f26dee0de4fe7bd3917b0eeb211c
     [1%]   [1/72]   9 53e13e5d66f560f3e2b04e74a099de0d
     [1%]   [1/72]  10 63914e33d55e1647962cf498030a7c16
     [2%]   [2/72]  11 76f8aa18dab86a06db6e70a0421dc28c
     [1%]   [1/72]  12 7cc969f65c7ec8d9db2fa57dcc51d323
     [2%]   [2/72]  13 926b734ea1992f8ee1f88ab967a93dac
     [1%]   [1/72]  14 9835a984a05caa405eb61faaa1546741
     [4%]   [3/72]  15 a0387c52951b3c2853740ef9cede1dec
     [4%]   [3/72]  16 a128136e4734a9f74c013356c773ece7
     [2%]   [2/72]  17 a22dbcbd09a98eacdd14ac7804a635dd
     [4%]   [3/72]  18 a568c03544f42dddf712bab3bfd562fd
     [5%]   [4/72]  19 b26767d88b31b8e1e88f61422786ec5e
     [5%]   [4/72]  20 b9c2a2d2339d471239b174dbbc6d8be2
     [2%]   [2/72]  21 bc4c952d089501afbfc8f7ff525da31c
     [2%]   [2/72]  22 cbc5fb89a0e5c9db5298b4b976ca7aca
     [1%]   [1/72]  23 d0e230454cb69aa01d4f86fc3a57b17f
     [4%]   [3/72]  24 df78b016db710335918cac342ae2a21d
     [2%]   [2/72]  25 e68303a089d42a09a9545cb48f3ff7a6
     [6%]   [5/72]  26 ef97014f01ea59c1ef337fe51a4d0331
     [4%]   [3/72]  27 f0ee042be143fcba78041fc2f69c0aa7
 ```
