# Bookmark

书签管理系统服务

** 截屏功能必须有！**

<img width="1487" alt="image" src="https://github.com/cute-angelia/bookmark/assets/26561606/f008051a-bc2e-4854-8aa9-ae1c256166ec">


### 使用说明

docker 启动 && 配置 chrome 扩展

1. docker 

    ```bash
    docker pull ghcr.io/cute-angelia/bookmark:latest
    
    docker run -d --name bookmark -v youdatapath:/app/data -p 38112:38112 --log-opt max-size=10m ghcr.io/cute-angelia/bookmark:latest
    ```

2. chrome extension 地址

    [✅chrome-bookmark](https://github.com/cute-angelia/chrome-bookmark)

 

