#! /bin/env sh
export CGO_ENABLED=0
go test -c --cover .
go test -c -o uapi_x86_64.test .
GOARCH=386 go test -c -o uapi_x86.test .
GOARCH=mips go test -c -o uapi_mips32.test .
GOARCH=mipsle go test -c -o uapi_mips32le.test .
GOARCH=mips64 go test -c -o uapi_mips64.test .
GOARCH=arm GOARM=6 go test -c -o uapi_arm32.test .
GOARCH=arm64 go test -c -o uapi_aarch64.test .
#GOARCH=riscv go test -c -o uapi_riscv.test .
GOARCH=riscv64 go test -c -o uapi_riscv64.test .

