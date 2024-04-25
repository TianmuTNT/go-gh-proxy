# go-gh-proxy
GitHub反向代理加速项目，使用Golang重写，原储存库https://github.com/hunshcn/gh-proxy
## 基本信息
默认端口3426，根目录下的ico和html为网页前端，json为配置文件
## 配置文件
这是一个配置文件的示例

```
{
    "white_list": [],
    "black_list": [],
    "size_limit": 1073741824
}
```
它可以配置用户或仓库的黑白名单，设置最大文件下载大小（Byte）等

**这只是一个示例**
