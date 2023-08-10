# Bookmark

书签管理系统服务

** 截屏功能必须有！**

<img width="745" alt="image" src="https://github.com/cute-angelia/bookmark/assets/26561606/7edc6387-7c62-4094-a785-07dda4c14533">


### dev

```shell
cd cmd/bookmark && go run main.go

open http://127.0.0.1:38112

username:admin
password:admin123

```

### docker

```bash
docker pull ghcr.io/cute-angelia/bookmark:latest

docker run -d --name bookmark -v youdatapath:/app/data -p 38112:38112 --log-opt max-size=10m ghcr.io/cute-angelia/bookmark:latest
```

### 其他配置

```shell
# 代理 基本不需要配置
PROXYADDR = socks5://XXXX.XXXX.XXXX.XXXX:8023
```

### chrome extension

[✅chrome-bookmark](https://github.com/cute-angelia/chrome-bookmark)
