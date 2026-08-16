# BENZHI 评测项目说明

## 项目用途

`task028888-oncall` 是一个纯 Go 命令行值班轮换调度器。它按日期范围、工程师名单、起始位置、节假日和工程师 blackout 配置生成值班安排，并输出公平性统计。

## 标准构建、运行和测试命令

在本目录执行：

```sh
GOTOOLCHAIN=local go build ./...
GOTOOLCHAIN=local go run . --smoke-test
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go run . --roster alice,bob --start 2026-03-02 --end 2026-03-04
```

`--smoke-test` 不依赖外部服务，执行后会自行退出。

## BENZHI Docker 构建

使用评测专用 `benzhi.Dockerfile`，不要替换为项目自带 Dockerfile：

```sh
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh task028888-oncall-benzhi linux/amd64
./build_benzhi_docker.sh task028888-oncall-benzhi linux/arm64
docker run --rm -it task028888-oncall-benzhi bash
```

脚本第一个参数是镜像名，第二个参数是平台，默认分别为 `my-project` 和 `linux/amd64`。
