module github.com/longhorn/go-spdk-helper

go 1.22.0

toolchain go1.23.1

require (
	github.com/c9s/goprocinfo v0.0.0-20210130143923-c95fcf8c64a8
	github.com/longhorn/go-common-libs v0.0.0-20240907130740-7060fefb5bda
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.3
	github.com/urfave/cli v1.22.15
	golang.org/x/sys v0.25.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
)

replace github.com/longhorn/go-common-libs v0.0.0-20240720044518-32fc527fe868 => github.com/derekbit/go-common-libs v0.0.0-20240726044555-d6c576d61bf3
